---
description: SAIA (Academic Cloud Hessen) models available via the zot-saia-plugin.
---

# SAIA Academic Cloud Models

This plugin registers the SAIA provider with 6 models hosted on the Academic Cloud Hessen infrastructure.

## Available Models

| Model ID | Name | Context | Reasoning |
|----------|------|---------|-----------|
| `saia/glm-4.7` | GLM 4.7 | 128K | ✅ |
| `saia/qwen3.5-397b-a17b` | Qwen 3.5 397B | 128K | ✅ |
| `saia/qwen3.5-122b-a10b` | Qwen 3.5 122B | 128K | ✅ |
| `saia/devstral-2-123b-instruct-2512` | DevStral 2 123B | 128K | ✅ |
| `saia/openai-gpt-oss-120b` | GPT-OSS 120B | 128K | ✅ |
| `saia/qwen3.6-27b` | Qwen 3.6 27B | 128K | ✅ |

## Usage

```bash
# List available models
zot --list-models | grep saia

# Switch to a SAIA model
zot --provider saia --model glm-4.7

# Large reasoning model
zot --provider saia --model qwen3.5-397b-a17b

# Lightweight model for quick tasks
zot --provider saia --model qwen3.6-27b
```

With the extension installed, use the slash command:
```
/saia-models
```

## API Key

The API key is resolved from `auth.json` (`saia` key), `$SAIA_API_KEY` environment variable, or `/login`.

Set it via:
```bash
export SAIA_API_KEY="your-key"
```

Or add to `$ZOT_HOME/auth.json`:
```json
{
  "saia": { "type": "api_key", "key": "your-key" }
}
```
