---
description: SAIA (Academic Cloud Hessen) models available via the zot-saia-plugin.
---

# SAIA Academic Cloud Models

This plugin registers the SAIA provider with 14 models hosted on the Academic Cloud Hessen infrastructure. The catalog auto-syncs with the live SAIA API (cached via L0/L1).

## Available Models

| Model ID | Name | Ctx | Out | Reasoning | Attach | Category |
|----------|------|-----|-----|-----------|--------|----------|
| `saia/qwen3.5-397b-a17b` | Qwen 3.5 397B A17B | 256K | 32K | ✅ | 🖼 | reasoning |
| `saia/qwen3.8-27b` | Qwen 3.8 27B | 262K | 32K | ✅ | — | reasoning |
| `saia/qwen3-coder-next` | Qwen 3 Coder Next | 256K | 16K | — | — | coder |
| `saia/devstral-2-123b-instruct-2512` | Devstral 2 123B Instruct 2512 | 256K | 16K | — | — | agentic |
| `saia/glm-5.3-flash` | GLM 5.3 Flash | 1M | 32K | ✅ | 🖼 | agentic |
| `saia/mistral-medium-3.5-128b` | Mistral Medium 3.5 128B | 256K | 8K | — | — | agentic |
| `saia/qwen3.6-35b-a3b` | Qwen 3.6 35B A3B | 262K | 16K | ✅ | 🖼 | agentic |
| `saia/openai-gpt-oss-120b` | GPT OSS 120B | 128K | 8K | ✅ | — | large-context |
| `saia/qwen3-omni-30b-a3b-instruct` | Qwen 3 Omni 30B A3B Instruct | 256K | 4K | — | 🖼 | vision |
| `saia/apertus-70b-instruct-2509` | Apertus 70B Instruct 2509 | 65K | 8K | — | — | general |
| `saia/deepseek-v4-flash-0731` | DeepSeek V4 Flash 0731 | 1M | 32K | ✅ | — | general |
| `saia/gemma-4-31b-it` | Gemma 4 31B Instruct | 256K | 8K | — | 🖼 | general |
| `saia/meta-llama-3.1-8b-instruct` | Llama 3.1 8B Instruct | 128K | 4K | — | — | general |
| `saia/qwen3-30b-a3b-instruct-2507` | Qwen 3 30B A3B Instruct 2507 | 256K | 16K | — | — | general |

## Quick Switch

```bash
# Best for agentic coding
zot --provider saia --model glm-5.3-flash

# Flagship reasoning (long context)
zot --provider saia --model qwen3.5-397b-a17b

# Code-specialized
zot --provider saia --model qwen3-coder-next

# Fast & lightweight
zot --provider saia --model deepseek-v4-flash-0731

# Vision (image input)
zot --provider saia --model qwen3.6-35b-a3b

# Budget (cheapest)
zot --provider saia --model meta-llama-3.1-8b-instruct
```

With the extension installed, use the slash commands:
```
/saia-models        # list models & usage
/saia-models refresh# force a fresh catalog refresh, then list
/saia-sync          # refresh model catalog from the live API (cached)
/saia-sync force    # bypass the cache and force a fresh refresh
/saia-cache         # show L0/L1 cache stats
/saia-cache clear   # clear L0 + L1 cache
```

## Caching

The plugin caches the model catalog (not the chat responses) using two optional tiers:

- **L0** — in-memory, 5-minute TTL (`SAIA_CACHE_L0=false` to disable)
- **L1** — disk at `~/.cache/saia/`, 24-hour TTL (`SAIA_CACHE_L1=false` to disable)

On a cache hit the catalog is used directly; on a miss the live API is queried and the result is cached.

## Rate Limits

SAIA enforces: **30 requests/min · 200/hour · 1,000/day · 3,000/month**

Check remaining quota at [SAIA dashboard](https://chat-ai.academiccloud.de).

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
