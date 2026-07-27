# zot-saia-plugin
> SAIA (Academic Cloud Hessen) provider for [zot](https://zot.sh)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A zot extension + models configuration that auto-registers all **SAIA Academic Cloud** models as a custom provider — no manual `--base-url` flags needed.

Ported from [pi-saia-plugin](https://github.com/tobias-weiss-ai-xr/pi-saia-plugin) following the same pattern.

## Features

- **Auto-registration** — Drop `models.json` into `$ZOT_HOME/` or your project to add all 6 SAIA models
- **Slash command** — `/saia-models` lists available models and usage
- **Skill included** — `SKILL.md` documents models, API key setup, and examples
- **OpenAI-compatible** — Uses standard OpenAI completions API

## Installation

### Option A: Install extension (recommended)

```bash
zot ext install /path/to/zot-saia-plugin
```

### Option B: Manual setup

```bash
# Copy models.json to your zot home
cp models.json ~/.local/state/zot/models.json

# Copy skill to your skills directory
cp skills/saia-models.md ~/.local/state/zot/skills/
```

Then restart zot:

```bash
zot
```

### API Key

Set your API key via environment variable or `auth.json`:

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
├── README.md               # This file
├── LICENSE                 # MIT
├── extension.json          # Zot extension manifest
├── models.json             # Custom provider + model definitions
├── src/
│   └── saia_extension.py   # Extension subprocess (slash commands)
└── skills/
    └── saia-models.md      # Model documentation skill
```

## Architecture

This plugin maps the pi package pattern to zot's extension model:

| pi concept | zot equivalent |
|---|---|
| `pi.registerProvider()` in TypeScript | `models.json` in `$ZOT_HOME/` |
| Extension (in-process TS) | Extension (subprocess + JSON-RPC, any language) |
| `SKILL.md` | `SKILL.md` (Agent Skills standard) |
| `pi install` | `zot ext install` |

- **models.json**: Defines the `saia` provider with its OpenAI-compatible base URL and all 6 models. Placed in `$ZOT_HOME/` so zot discovers it on startup.
- **Extension** (`src/saia_extension.py`): A lightweight Python subprocess that registers the `/saia-models` slash command for quick model reference.
- **Skill** (`skills/saia-models.md`): Provides user-facing documentation of available models, usage examples, and API key setup.

## License

MIT
