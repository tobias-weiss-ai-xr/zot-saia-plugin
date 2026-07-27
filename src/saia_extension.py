#!/usr/bin/env python3
"""
zot-saia-plugin: SAIA (Academic Cloud Hessen) extension for zot.

Registers the /saia-models slash command that displays available
SAIA models, usage examples, and API key setup instructions.

Lifecycle: hello → hello_ack → register_command → ready → runtime → shutdown
Wire format: newline-delimited JSON over stdin/stdout.
"""

import json
import sys

MODELS = [
    {
        "id": "glm-4.7",
        "name": "GLM 4.7",
        "ctx": "128K",
    },
    {
        "id": "qwen3.5-397b-a17b",
        "name": "Qwen 3.5 397B",
        "ctx": "128K",
    },
    {
        "id": "qwen3.5-122b-a10b",
        "name": "Qwen 3.5 122B",
        "ctx": "128K",
    },
    {
        "id": "devstral-2-123b-instruct-2512",
        "name": "DevStral 2 123B",
        "ctx": "128K",
    },
    {
        "id": "openai-gpt-oss-120b",
        "name": "GPT-OSS 120B",
        "ctx": "128K",
    },
    {
        "id": "qwen3.6-27b",
        "name": "Qwen 3.6 27B",
        "ctx": "128K",
    },
]


def emit(obj: dict) -> None:
    """Write one JSON line to stdout and flush."""
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def build_models_prompt() -> str:
    """Build a prompt string listing all SAIA models."""
    lines = [
        "## SAIA Academic Cloud — Available Models",
        "",
        "| Model ID | Name | Context | Reasoning |",
        "|----------|------|---------|-----------|",
    ]
    for m in MODELS:
        lines.append(f"| saia/{m['id']} | {m['name']} | {m['ctx']} | ✅ |")
    lines.extend([
        "",
        "### Usage",
        "",
        "Switch to a SAIA model:",
        "  /model saia/glm-4.7",
        "",
        "Large reasoning model:",
        "  /model saia/qwen3.5-397b-a17b",
        "",
        "Lightweight model for quick tasks:",
        "  /model saia/qwen3.6-27b",
        "",
        "### API Key",
        "",
        "Set via environment variable:",
        '  export SAIA_API_KEY="your-key"',
        "",
        "Or add to auth.json (saia key), or use /login.",
        "",
    ])
    return "\n".join(lines)


def main() -> None:
    # Phase 1: hello handshake
    emit({
        "type": "hello",
        "name": "zot-saia-plugin",
        "version": "1.0.0",
        "capabilities": ["commands"],
    })

    for line in sys.stdin:
        msg = json.loads(line)
        msg_type = msg.get("type", "")

        if msg_type == "hello_ack":
            # Phase 2: register commands
            emit({
                "type": "register_command",
                "name": "saia-models",
                "description": "list SAIA Academic Cloud models and usage",
            })
            emit({"type": "ready"})

        elif msg_type == "command_invoked":
            # Phase 3: handle slash command
            emit({
                "type": "command_response",
                "id": msg["id"],
                "action": "prompt",
                "prompt": build_models_prompt(),
            })

        elif msg_type == "shutdown":
            emit({"type": "shutdown_ack"})
            break


if __name__ == "__main__":
    main()
