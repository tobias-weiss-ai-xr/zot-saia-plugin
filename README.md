# zot-saia-plugin
> SAIA (Academic Cloud Hessen) provider for [zot](https://zot.sh)

> ⚠️ **Note:** Active development takes place on [GitHub](https://github.com/tobias-weiss-ai-xr/zot-saia-plugin). Any other hosted copies are **legacy mirrors** — synced periodically but not actively maintained there.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A self-contained Go extension for zot that auto-registers all **SAIA Academic Cloud** models as a custom provider — no manual `--base-url` flags needed.

Ported from [pi-saia-plugin](https://github.com/tobias-weiss-ai-xr/pi-saia-plugin).

**Other Platforms:**
- [opencode-saia-plugin](https://github.com/tobias-weiss-ai-xr/opencode-saia-plugin) — SAIA provider for OpenCode
- [pi-saia-plugin](https://github.com/tobias-weiss-ai-xr/pi-saia-plugin) — SAIA provider for pi coding agent

## Features

- **Self-contained Go binary** — single static executable, no runtime dependencies
- **Auto-registration** — `models.json` adds all 16 SAIA models on startup
- **Model auto-update** — fetches the live model catalog on load and syncs with the API (cached)
- **L0/L1 caching** — optional in-memory + disk cache, same semantics as the pi/opencode plugins
- **Slash commands** — `/saia-models`, `/saia-sync`, `/saia-cache`
- **Skill included** — `SKILL.md` documents models, API key setup, and examples
- **OpenAI-compatible** — Uses standard OpenAI completions API

## Caching

The plugin ships with optional **L0 (in-memory)** and **L1 (disk)** caching, mirroring the
behaviour of the pi and opencode plugins:

- **L0** — in-memory, 5-minute TTL (fastest)
- **L1** — disk (`~/.cache/saia/`), 24-hour TTL
- Lookup priority: L0 → L1 → live API; an L1 hit primes L0

| Environment variable | Default | Effect |
|---|---|---|
| `SAIA_CACHE_L0=false` | enabled | Disables the in-memory tier |
| `SAIA_CACHE_L1=false` | enabled | Disables the disk tier |

```bash
/saia-cache        # show cache stats
/saia-cache clear  # clear L0 + L1
```

## Model auto-update

The plugin refreshes the model catalog from the live SAIA `/v1/models` endpoint:

- Automatically on plugin load (cache-aware, safe without an API key)
- On demand with `/saia-sync` (or `/saia-sync force` to bypass the cache)
- `/saia-models refresh` also forces a refresh

New models are added, removed models are dropped, and reasoning/attachment flags are
re-derived from the live API (`output`/`input` modalities).

## Quick Start

### Build

```bash
git clone https://github.com/tobias-weiss-ai-xr/zot-saia-plugin.git
cd zot-saia-plugin
./build.sh
```

Or manually:

```bash
CGO_ENABLED=0 go build -o zot-saia-plugin ./cmd/zot-saia-plugin
```

### Install into zot

```bash
zot ext install .
```

Then restart zot and use:

```
/saia-models        # list models & usage
/saia-sync          # refresh model catalog (cached)
/saia-sync force    # force a fresh refresh
/saia-cache         # show cache stats
/saia-cache clear   # clear the cache
```

### API Key

```bash
export SAIA_API_KEY="your-key"
```

Or add to `$ZOT_HOME/auth.json`:
```json
{
  "saia": { "type": "api_key", "key": "your-key" }
}
```

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

# Use a SAIA model
zot --provider saia --model glm-4.7

# With the extension installed, use the slash command
/saia-models
```

## Plugin Structure

```
zot-saia-plugin/
├── cmd/zot-saia-plugin/
│   └── main.go              # Extension binary (uses zot Go SDK)
├── cache/
│   ├── cache.go             # L0/L1 caching
│   └── cache_test.go        # cache tests
├── models/
│   ├── models.go            # Model catalog + prompt builder
│   ├── sync.go              # Model auto-update from live API
│   ├── sync_test.go         # sync tests
│   └── models_test.go        # catalog tests
├── skills/
│   └── saia-models.md      # Model documentation skill
├── models.json              # Custom provider + model definitions
├── extension.json           # Zot extension manifest
├── go.mod                   # Go module
├── build.sh                 # Build script (static binary)
├── LICENSE
└── README.md
```

## Architecture

| pi concept | zot equivalent |
|---|---|
| `pi.registerProvider()` in TypeScript | `models.json` in `$ZOT_HOME/` |
| Extension (in-process TS) | Go binary via `ext` SDK (subprocess + JSON-RPC) |
| `SKILL.md` | `SKILL.md` (Agent Skills standard) |
| `pi install` | `zot ext install` |

## License

MIT
