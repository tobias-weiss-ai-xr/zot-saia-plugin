# zot-saia-plugin
> SAIA (Academic Cloud Hessen) provider for [zot](https://zot.sh)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A self-contained Go extension for zot that auto-registers all **SAIA Academic Cloud** models as a custom provider — no manual `--base-url` flags needed.

Ported from [pi-saia-plugin](https://github.com/tobias-weiss-ai-xr/pi-saia-plugin).

## Features

- **Self-contained Go binary** — single static executable, no runtime dependencies
- **Auto-registration** — `models.json` adds all 6 SAIA models on startup
- **Slash command** — `/saia-models` lists available models and usage
- **Skill included** — `SKILL.md` documents models, API key setup, and examples
- **OpenAI-compatible** — Uses standard OpenAI completions API

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
/saia-models
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
