# Next steps

These are not part of the current Linux desktop game. The rules engine in `internal/game` is independent of the UI so each item can build on it.

## 1. MCP Server

Expose the rules engine as an MCP server so agents can play or inspect a game.

- Tools for the current position, legal plays, applying a play, and asking a bot (easy, intermediate, hard, AI) to move
- Reuse `internal/game` and `internal/bot` without linking the Ebitengine UI
- Keep the server a separate command so the desktop game stays a normal binary

## 2. Ollama Support

Done. Settings has OpenRouter / Ollama provider chips, base URL, model field, Check (lists `/api/tags` and cycles installed models), and offline guidance when the daemon is down. Choose Opponent **AI** to play with the saved provider.

## 3. Multiple OS support

Build and package the Ebitengine game for macOS, Windows, and Unix (BSD), not only Linux.

- Taskfile targets per OS, plus a note of the native graphics and audio libraries each one needs
- Verify windowing, OpenGL, and audio on each target
- Keep saving settings in the per-OS user config directory (`os.UserConfigDir`)
