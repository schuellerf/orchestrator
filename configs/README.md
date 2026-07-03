# Gjoll sandbox Terraform configs

Use the unified libvirt template in the gjoll repository:

```
../gjoll/examples/fedora-libvirt
```

Proxy mode is selected automatically from `orchestrator.yaml`:

| Setting | `proxy_mode` |
|---------|--------------|
| `llm_base_url` set (local LLM) | `local-llm` |
| `llm_base_url: ""` + `vertex_project_id` | `vertex` (default cloud) |
| `gjoll_env` path contains `anthropic` | `anthropic` |

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

Set `gjoll_env` to a path containing `anthropic`, e.g. copy
`sandbox-anthropic-api.tf.example` to `sandbox-anthropic.tf`, or use the unified
template with `proxy_mode=anthropic` via orchestrator (path must contain `anthropic`).

```yaml
gjoll_env: "./configs/sandbox-anthropic.tf"
llm_base_url: ""
anthropic_key_file: "~/.anthropic/api_key"
agent_backend: "claude-code"
```

Agent installation and configuration are handled by the orchestrator at runtime, not in the Terraform `init_script`.
