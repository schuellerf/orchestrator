# Gjoll sandbox Terraform configs

Use the unified templates in the gjoll repository:

| Backend | Path |
|---------|------|
| Libvirt (local KVM) | `../gjoll/examples/fedora-libvirt` |
| AWS EC2 | `../gjoll/examples/fedora-aws` |

The orchestrator sets `TF_VAR_proxy_mode`, `TF_VAR_agent_backend`, and (for AWS) `TF_VAR_ami_id` from `orchestrator.yaml` before `gjoll up`. Agent install runs in the environment `init_script`.

| Setting | `proxy_mode` |
|---------|--------------|
| `llm_base_url` set (local LLM) | `local-llm` |
| `llm_base_url: ""` + `vertex_project_id` | `vertex` (default cloud) |
| `gjoll_env` path contains `anthropic` | `anthropic` |

| Setting | `agent_backend` (→ `TF_VAR_agent_backend`) |
|---------|---------------------------------------------|
| `agent_backend: "opencode"` | `opencode` |
| `agent_backend: "claude-code"` | `claude-code` |

## Vertex AI (default cloud)

```yaml
gjoll_env: "../gjoll/examples/fedora-libvirt"
llm_base_url: ""
vertex_project_id: "your-gcp-project"
vertex_region: "us-east5"   # optional, default us-east5
agent_backend: "opencode"
```

Requires GCP Application Default Credentials on the host (`gcloud auth application-default login`).

## Local LLM (Ollama / LM Studio)

```yaml
gjoll_env: "../gjoll/examples/fedora-libvirt"
llm_base_url: "http://127.0.0.1:11434/v1"
llm_model: "your-model-id"
agent_backend: "opencode"
```

## Anthropic API (direct)

Use the unified template with a `gjoll_env` path containing `anthropic` (symlink or copy):

```bash
ln -s ../gjoll/examples/fedora-libvirt configs/sandbox-anthropic
```

```yaml
gjoll_env: "./configs/sandbox-anthropic"
llm_base_url: ""
anthropic_key_file: "~/.anthropic/api_key"
agent_backend: "claude-code"
```

## AWS EC2

```yaml
gjoll_env: "../gjoll/examples/fedora-aws"
llm_base_url: ""
vertex_project_id: "your-gcp-project"
vertex_region: "us-east5"   # optional, default us-east5
agent_backend: "opencode"
aws_ami_id: "ami-0edf1d45580ac3fa3"   # optional, Fedora 43 us-east-1 default
```

Requires AWS credentials on the host (`aws configure` or instance role). For Vertex AI, also requires GCP Application Default Credentials (`gcloud auth application-default login`).
