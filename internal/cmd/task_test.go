package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/drellahq/orchestrator/internal/config"
	"github.com/drellahq/orchestrator/internal/task"
)

func TestParseVarFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
		want  map[string]string
	}{
		{
			name:  "nil flags",
			flags: nil,
			want:  nil,
		},
		{
			name:  "empty flags",
			flags: []string{},
			want:  nil,
		},
		{
			name:  "single var",
			flags: []string{"PROFILE_PR=42"},
			want:  map[string]string{"PROFILE_PR": "42"},
		},
		{
			name:  "multiple vars",
			flags: []string{"PROFILE_REPO=org/repo", "PROFILE_PR=42"},
			want:  map[string]string{"PROFILE_REPO": "org/repo", "PROFILE_PR": "42"},
		},
		{
			name:  "value with equals sign",
			flags: []string{"KEY=a=b=c"},
			want:  map[string]string{"KEY": "a=b=c"},
		},
		{
			name:  "no equals sign is skipped",
			flags: []string{"NOEQUALS"},
			want:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVarFlags(tt.flags)
			if tt.want == nil {
				if got != nil {
					t.Errorf("parseVarFlags(%v) = %v, want nil", tt.flags, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseVarFlags(%v) has %d entries, want %d", tt.flags, len(got), len(tt.want))
			}
			for k, wantV := range tt.want {
				if got[k] != wantV {
					t.Errorf("parseVarFlags(%v)[%q] = %q, want %q", tt.flags, k, got[k], wantV)
				}
			}
		})
	}
}

func TestWriteBudgetJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			MaxBudgetUSD:      100,
			WarnBudgetUSD:     30,
			CriticalBudgetUSD: 50,
		},
	}

	if err := writeBudgetJSON(cfg, dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "budget.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got config.AgentConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if got.MaxBudgetUSD != 100 {
		t.Errorf("max_budget_usd = %v, want 100", got.MaxBudgetUSD)
	}
	if got.WarnBudgetUSD != 30 {
		t.Errorf("warn_budget_usd = %v, want 30", got.WarnBudgetUSD)
	}
	if got.CriticalBudgetUSD != 50 {
		t.Errorf("critical_budget_usd = %v, want 50", got.CriticalBudgetUSD)
	}
	if got.OpenCodeBashTimeout != "" {
		t.Errorf("opencode_bash_timeout = %q, want empty", got.OpenCodeBashTimeout)
	}
}

func TestWriteBudgetJSON_zeroes(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}

	if err := writeBudgetJSON(cfg, dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "budget.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got config.AgentConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if got.MaxBudgetUSD != 0 {
		t.Errorf("max_budget_usd = %v, want 0", got.MaxBudgetUSD)
	}
	if got.WarnBudgetUSD != 0 {
		t.Errorf("warn_budget_usd = %v, want 0", got.WarnBudgetUSD)
	}
	if got.CriticalBudgetUSD != 0 {
		t.Errorf("critical_budget_usd = %v, want 0", got.CriticalBudgetUSD)
	}
	if got.OpenCodeBashTimeout != "" {
		t.Errorf("opencode_bash_timeout = %q, want empty", got.OpenCodeBashTimeout)
	}
}

func TestResolveTaskSource(t *testing.T) {
	outputDir := t.TempDir()
	td, err := task.Create(outputDir, "my-task")
	if err != nil {
		t.Fatal(err)
	}
	if err := td.SaveMetadata("my-task", "desc", "", time.Now()); err != nil {
		t.Fatal(err)
	}

	if _, _, ok := resolveTaskSource(td, "org/tasks", 42, ""); !ok {
		t.Error("expected ok from explicit source")
	}
	if repo, num, ok := resolveTaskSource(td, "org/tasks", 42, ""); !ok || repo != "org/tasks" || num != 42 {
		t.Errorf("got %q %d %v", repo, num, ok)
	}

	if err := td.SaveSource("org/tasks", 99); err != nil {
		t.Fatal(err)
	}
	if repo, num, ok := resolveTaskSource(td, "", 0, "fallback/tasks"); !ok || repo != "org/tasks" || num != 99 {
		t.Errorf("from state: got %q %d %v", repo, num, ok)
	}

	td2, err := task.Create(outputDir, "other-task")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := resolveTaskSource(td2, "", 0, ""); ok {
		t.Error("expected not ok without source")
	}
}

func TestSetupGjollProxyVars(t *testing.T) {
	clearGjollTFVars(t)

	empty := ""
	cfg := &config.Config{
		SandboxBackend:  "gjoll",
		LLMBaseURL:      &empty,
		VertexProjectID: "proj-123",
		VertexRegion:    config.DefaultVertexRegion,
		GjollEnv:        "../gjoll/examples/fedora-libvirt",
	}

	if err := setupGjollProxyVars(cfg, "opencode"); err != nil {
		t.Fatalf("setupGjollProxyVars() error: %v", err)
	}
	if got := os.Getenv("TF_VAR_agent_backend"); got != "opencode" {
		t.Fatalf("TF_VAR_agent_backend = %q, want opencode", got)
	}
	if got := os.Getenv("TF_VAR_proxy_mode"); got != "vertex" {
		t.Fatalf("TF_VAR_proxy_mode = %q, want vertex", got)
	}
	if got := os.Getenv("TF_VAR_proxy_port"); got != "18080" {
		t.Fatalf("TF_VAR_proxy_port = %q, want 18080", got)
	}
	if got := os.Getenv("TF_VAR_vertex_project_id"); got != "proj-123" {
		t.Fatalf("TF_VAR_vertex_project_id = %q, want proj-123", got)
	}
}

func TestSetupGjollProxyVarsLocalLLM(t *testing.T) {
	clearGjollTFVars(t)

	url := "http://127.0.0.1:11434/v1"
	cfg := &config.Config{
		SandboxBackend: "gjoll",
		LLMBaseURL:     &url,
		GjollEnv:       "../gjoll/examples/fedora-libvirt",
	}

	if err := setupGjollProxyVars(cfg, "claude-code"); err != nil {
		t.Fatalf("setupGjollProxyVars() error: %v", err)
	}
	if got := os.Getenv("TF_VAR_agent_backend"); got != "claude-code" {
		t.Fatalf("TF_VAR_agent_backend = %q, want claude-code", got)
	}
	if got := os.Getenv("TF_VAR_proxy_mode"); got != "local-llm" {
		t.Fatalf("TF_VAR_proxy_mode = %q, want local-llm", got)
	}
	if got := os.Getenv("TF_VAR_llm_host_port"); got != "11434" {
		t.Fatalf("TF_VAR_llm_host_port = %q, want 11434", got)
	}
	if got := os.Getenv("TF_VAR_llm_proxy_port"); got != "11434" {
		t.Fatalf("TF_VAR_llm_proxy_port = %q, want 11434", got)
	}
}

func TestSetupGjollProxyVarsVertexRequiresProject(t *testing.T) {
	clearGjollTFVars(t)

	empty := ""
	cfg := &config.Config{
		SandboxBackend: "gjoll",
		LLMBaseURL:     &empty,
		GjollEnv:       "../gjoll/examples/fedora-libvirt",
	}

	if err := setupGjollProxyVars(cfg, "claude-code"); err == nil {
		t.Fatal("expected error when vertex_project_id is missing for claude-code + vertex")
	}
}

func TestSetupGjollProxyVarsSkipsPodman(t *testing.T) {
	clearGjollTFVars(t)

	cfg := &config.Config{SandboxBackend: "podman"}
	if err := setupGjollProxyVars(cfg, "opencode"); err != nil {
		t.Fatalf("setupGjollProxyVars() error: %v", err)
	}
	if got := os.Getenv("TF_VAR_agent_backend"); got != "" {
		t.Fatalf("TF_VAR_agent_backend = %q, want unset for podman", got)
	}
}

func clearGjollTFVars(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"TF_VAR_agent_backend",
		"TF_VAR_proxy_mode",
		"TF_VAR_llm_host_port",
		"TF_VAR_llm_proxy_port",
		"TF_VAR_vertex_project_id",
		"TF_VAR_vertex_region",
		"TF_VAR_proxy_port",
		"TF_VAR_anthropic_key_file",
	} {
		t.Setenv(key, "")
	}
}
