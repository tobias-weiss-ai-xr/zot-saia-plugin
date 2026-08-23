---
description: SAIA (Academic Cloud Hessen) models available via the zot-saia-plugin.
---

# SAIA Academic Cloud Models

This plugin registers the SAIA provider with 16 models hosted on the Academic Cloud Hessen infrastructure. The catalog auto-syncs with the live SAIA API (cached via L0/L1).

## Available Models

| Model ID | Name | Ctx | Out | Reasoning | Attach | Category |
|----------|------|-----|-----|-----------|--------|----------|
| `saia/qwen3.5-397b-a17b` | Qwen 3.5 397B | 128K | 32K | ✅ | 🖼 | reasoning |
| `saia/qwen3.5-122b-a10b` | Qwen 3.5 122B | 128K | 32K | ✅ | 🖼 | reasoning |
| `saia/glm-4.7` | GLM 4.7 | 128K | 16K | — | — | general |
| `saia/qwen3-30b-a3b-instruct-2507` | Qwen 3 30B | 128K | 16K | — | — | general |
| `saia/devstral-2-123b-instruct-2512` | DevStral 2 123B | 128K | 16K | — | — | agentic |
| `saia/mistral-medium-3.5-128b` | Mistral Medium 3.5 128B | 128K | 8K | — | — | agentic |
| `saia/qwen3.6-35b-a3b` | Qwen 3.6 35B | 128K | 16K | — | 🖼 | agentic |
| `saia/qwen3-coder-next` | Qwen 3 Coder Next | 128K | 16K | — | — | coder |
| `saia/openai-gpt-oss-120b` | GPT-OSS 120B | 128K | 8K | — | — | large-context |
| `saia/medgemma-27b-it` | MedGemma 27B | 32K | 4K | — | 🖼 | medical |
| `saia/qwen3-omni-30b-a3b-instruct` | Qwen 3 Omni 30B | 32K | 4K | — | 🖼 | vision |
| `saia/deepseek-v4-flash-0731` | DeepSeek V4 Flash | 128K | 16K | — | — | general |
| `saia/qwen3.6-27b` | Qwen 3.6 27B | 128K | 16K | — | — | general |
| `saia/gemma-4-31b-it` | Gemma 4 31B | 128K | 8K | — | 🖼 | general |
| `saia/apertus-70b-instruct-2509` | Apertus 70B | 128K | 8K | — | — | general |
| `saia/meta-llama-3.1-8b-instruct` | Meta Llama 3.1 8B | 128K | 4K | — | — | general |

## Quick Switch

```bash
# Best for agentic coding
zot --provider saia --model glm-4.7

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
