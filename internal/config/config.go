package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/drellahq/orchestrator/internal/agent"
	"github.com/goccy/go-yaml"
)

// DefaultOpenCodeBashTimeout is the default bash tool timeout for OpenCode sandboxes.
const DefaultOpenCodeBashTimeout = 3 * time.Hour

// AgentConfig holds per-agent budget limits.
type AgentConfig struct {
	MaxBudgetUSD        float64 `yaml:"max-budget-usd" json:"max_budget_usd"`
	WarnBudgetUSD       float64 `yaml:"warn-budget-usd" json:"warn_budget_usd"`
	CriticalBudgetUSD   float64 `yaml:"critical-budget-usd" json:"critical_budget_usd"`
	OpenCodeBashTimeout string  `yaml:"opencode_bash_timeout" json:"opencode_bash_timeout"`
}

// OpenCodeBashTimeoutDuration returns the configured OpenCode bash tool timeout,
// defaulting to DefaultOpenCodeBashTimeout when unset.
func (a AgentConfig) OpenCodeBashTimeoutDuration() (time.Duration, error) {
	if a.OpenCodeBashTimeout == "" {
		return DefaultOpenCodeBashTimeout, nil
	}
	d, err := time.ParseDuration(a.OpenCodeBashTimeout)
	if err != nil {
		return 0, fmt.Errorf("parsing agent.opencode_bash_timeout: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("agent.opencode_bash_timeout must be positive")
	}
	return d, nil
}

const DefaultLLMBaseURL = "http://127.0.0.1:1234/v1"

// DefaultVertexRegion is the default GCP region for Vertex AI proxy targets.
const DefaultVertexRegion = "us-east5"

// DefaultAWSAMIID is the default Fedora 43 x86_64 AMI in us-east-1.
const DefaultAWSAMIID = "ami-0edf1d45580ac3fa3"

// GjollCloudProxyPort is the in-VM port for gjoll credential proxies (vertex, anthropic).
const GjollCloudProxyPort = 18080

// GjollLocalLLMProxyPort is the in-VM port for the local-llm gjoll reverse proxy.
const GjollLocalLLMProxyPort = 11434

// DaemonConfig holds settings for the daemon polling loop.
type DaemonConfig struct {
	PollInterval      string   `yaml:"poll_interval"`
	AllowedCommenters []string `yaml:"allowed_commenters"`
	TasksRepo         string   `yaml:"tasks_repo"`
}

type Config struct {
	SlackWebhook string       `yaml:"slack_webhook"`
	OutputDir    string       `yaml:"output_dir"`
	BaseURL      string       `yaml:"base_url"`

	// Sandbox backend: "gjoll" (VMs) or "podman" (containers)
	SandboxBackend string `yaml:"sandbox_backend"`

	// Gjoll backend settings
	GjollEnv string `yaml:"gjoll_env"` // path to .tf file or directory for VM provisioning
	AWSAMIID string `yaml:"aws_ami_id"` // EC2 AMI for gjoll AWS templates

	// Vertex AI settings (gjoll proxy_mode=vertex)
	VertexProjectID string `yaml:"vertex_project_id"`
	VertexRegion    string `yaml:"vertex_region"`

	// Podman backend settings
	PodmanImage      string `yaml:"podman_image"`       // container image (e.g. "fedora:43")
	AnthropicKeyFile string `yaml:"anthropic_key_file"` // path to API key for cloud Anthropic

	// Agent backend: "claude-code" or "opencode"
	AgentBackend string `yaml:"agent_backend"`

	// LLMBaseURL points agents at a local OpenAI/Anthropic-compatible API (LM Studio).
	// nil = default LM Studio URL; explicit empty string disables local LLM.
	LLMBaseURL *string `yaml:"llm_base_url"`
	// LLMModel selects a model on the local API. When empty, the first model
	// from /v1/models is used.
	LLMModel string `yaml:"llm_model"`

	AllowedRepos []string     `yaml:"allowed_repos"`
	ProfilesRepo string       `yaml:"profiles_repo"`
	ProfilesDir  string       `yaml:"profiles_dir"`
	Agent        AgentConfig  `yaml:"agent"`
	Daemon       DaemonConfig `yaml:"daemon"`
}

// UsesLocalLLM reports whether agents should use a local LLM API instead of cloud credentials.
func (c *Config) UsesLocalLLM() bool {
	return c.LLMBaseURL != nil && *c.LLMBaseURL != ""
}

// LocalLLMBaseURL returns the configured local LLM base URL, or empty when disabled.
func (c *Config) LocalLLMBaseURL() string {
	if !c.UsesLocalLLM() {
		return ""
	}
	return *c.LLMBaseURL
}

// LocalLLMHostPort returns the TCP port from llm_base_url on the orchestrator host.
func (c *Config) LocalLLMHostPort() (int, error) {
	baseURL := c.LocalLLMBaseURL()
	if baseURL == "" {
		return 0, fmt.Errorf("local LLM is disabled")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return 0, fmt.Errorf("parsing llm_base_url: %w", err)
	}
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "https":
			return 443, nil
		default:
			return 80, nil
		}
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return 0, fmt.Errorf("invalid port in llm_base_url: %w", err)
	}
	return n, nil
}

// LocalLLMProxyPort returns the in-VM gjoll proxy port for local-llm mode.
func (c *Config) LocalLLMProxyPort() int {
	return GjollLocalLLMProxyPort
}

// GjollLLMBaseURL returns the Anthropic-compatible base URL agents use inside a gjoll VM.
// Requests go through the gjoll reverse proxy defined in the .tf proxies output.
func (c *Config) GjollLLMBaseURL() (string, error) {
	if !c.UsesLocalLLM() {
		return "", fmt.Errorf("local LLM is disabled")
	}
	return fmt.Sprintf("http://127.0.0.1:%d/v1", c.LocalLLMProxyPort()), nil
}

// GjollCloudLLMBaseURL returns the Anthropic-compatible base URL for gjoll cloud proxies inside the VM.
func GjollCloudLLMBaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/v1", GjollCloudProxyPort)
}

// AgentOptions returns agent backend options derived from this config.
// For gjoll sandboxes, the base URL targets the in-VM gjoll proxy port.
func (c *Config) AgentOptions() agent.Options {
	llmURL := c.LocalLLMBaseURL()
	if c.SandboxBackend == "gjoll" {
		if c.UsesLocalLLM() {
			if proxyURL, err := c.GjollLLMBaseURL(); err == nil {
				llmURL = proxyURL
			}
		} else if !strings.Contains(c.GjollEnv, "anthropic") {
			llmURL = GjollCloudLLMBaseURL()
		}
	}
	return agent.Options{LLMBaseURL: llmURL, LLMModel: c.LLMModel}
}

// ResolvedGjollProxyMode returns the proxy_mode value for unified gjoll libvirt templates.
func (c *Config) ResolvedGjollProxyMode() string {
	if c.UsesLocalLLM() {
		return "local-llm"
	}
	if strings.Contains(c.GjollEnv, "anthropic") {
		return "anthropic"
	}
	return "vertex"
}

// UsesVertexProxy reports whether gjoll sandboxes use the Vertex AI credential proxy.
func (c *Config) UsesVertexProxy() bool {
	return c.SandboxBackend == "gjoll" && !c.UsesLocalLLM() && !strings.Contains(c.GjollEnv, "anthropic")
}

// AnthropicKeyPath returns the API key file path for sandbox provisioning, or empty when using a local LLM.
func (c *Config) AnthropicKeyPath() string {
	if c.UsesLocalLLM() {
		return ""
	}
	return c.AnthropicKeyFile
}

// RepoAllowed reports whether repo is permitted by the AllowedRepos allowlist.
// Each entry may contain wildcards understood by path.Match (e.g. "org/*").
// An empty list denies all repos.
func (c *Config) RepoAllowed(repo string) bool {
	for _, pattern := range c.AllowedRepos {
		if matched, _ := path.Match(pattern, repo); matched {
			return true
		}
	}
	return false
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Defaults
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./tasks"
	}
	if cfg.SandboxBackend == "" {
		cfg.SandboxBackend = "gjoll"
	}
	if cfg.GjollEnv == "" {
		cfg.GjollEnv = "../gjoll/examples/fedora-libvirt"
	}
	if cfg.VertexRegion == "" {
		cfg.VertexRegion = DefaultVertexRegion
	}
	if cfg.AWSAMIID == "" {
		cfg.AWSAMIID = DefaultAWSAMIID
	}
	if cfg.PodmanImage == "" {
		cfg.PodmanImage = "fedora:43"
	}
	if cfg.AgentBackend == "" {
		cfg.AgentBackend = "opencode"
	}
	if cfg.LLMBaseURL == nil {
		defaultURL := DefaultLLMBaseURL
		cfg.LLMBaseURL = &defaultURL
	}
	if cfg.AnthropicKeyFile == "" && !cfg.UsesLocalLLM() {
		cfg.AnthropicKeyFile = "~/.anthropic/api_key"
	}
	if cfg.Agent.OpenCodeBashTimeout != "" {
		if _, err := cfg.Agent.OpenCodeBashTimeoutDuration(); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}
