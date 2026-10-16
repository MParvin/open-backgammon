# Backgammon

A desktop backgammon game for Linux. The board is drawn with [Ebitengine](https://ebitengine.org/). Rules, bots, and the window are separate packages so later work can add an MCP server, Ollama, and other operating systems without rewriting the game.

## Play

White moves first, toward the lower-right home board (points 1–6). Black moves toward the upper-right (points 19–24).

From the menu, pick an opponent and a match length (3–21 points):

| Opponent | How it plays |
| --- | --- |
| Human | Two people at one keyboard, taking turns |
| Easy | Softened search with evaluation noise and occasional blunders |
| Intermediate | Stronger static evaluation over more candidates |
| Hard | Top candidates scored with opponent-dice lookahead |
| AI | Asks OpenRouter or Ollama to choose a numbered legal play |

Matches use single / gammon / backgammon scoring and a doubling cube (Crawford rule). Offer a double with the Double button when it is legal; Take or Drop when the computer doubles.

### Themes and surrender

Open the hamburger menu for **Themes** and **Surrender**. The default **Persian Ornate** theme draws the real board plate from `assets/board/persian_ornate.jpg` (cleaned from the reference artwork), with live checkers, dice in the left well, and scores in the side plaques. Classic, Chinese, and Modern remain vector themes. Themes apply immediately and are saved in `~/.config/backgammon/config.json`. Surrender asks for confirmation, then awards the opponent the scored result for the current position × cube (or a Drop if a computer double is pending).

### Checker controls

- **Left-click the board** (or press **Space**) to roll when it is your turn.
- **Left-click** your checker: use the **larger** remaining die (or select it and show destinations if that die has more than one legal landing).
- **Right-click** your checker: use the **smaller** remaining die the same way. (Native window; no browser context menu.)
- **Click a highlighted destination** or **drag** after a selection when you need to pick among landings.
- **Middle-click** or **Ctrl+Z**: undo the last voluntary human move of the **current** turn.
- Unambiguous forced sequences (and empty rolls) play automatically.
- When your sequence is finished, **left-click** or press **Space** to end the turn. Undo is still available until then.

**Undo** restores a full snapshot: checkers, bar, hits, remaining dice, cube, and match score. A multi-die move and an automatic forced sequence are each one undo step. Undo is refused after the turn ends, after the AI has moved, or after a game result is recorded. The status line says so.

Matches score 1 / 2 / 3 for a single, gammon, or backgammon, multiplied by the cube. The Crawford game (the one after a player first reaches one point short of the target) is played without the cube. The computer offers and takes doubles from a separate equity estimate, not from pip count alone.

The right-hand tray is bearing off. Dice stay visible for the whole turn and clear when the turn ends. The AI never plays a move outside the legal list; if an LLM request fails, Intermediate moves instead and the status line says so.

## Scores and stats

Large pip counts for White and Black sit in the top-right of the board. They update after every move. Lower is closer to bearing off. Click the doubling cube (or your pip count, or press D) before you roll to offer a double.

Against a bot, finished matches are appended to `~/.config/backgammon/history.json`. The menu **Stats** page shows wins, losses, win rate, recent matches, and a win-rate chart over time. The file starts empty; nothing is invented.

## AI setup

In Settings, choose provider **OpenRouter** or **Ollama**.

OpenRouter:

```sh
export OPENROUTER_API_KEY=...
export OPENROUTER_MODEL=openai/gpt-4o-mini
```

Ollama (default URL `http://localhost:11434`):

```sh
export OLLAMA_HOST=http://localhost:11434
```

Settings are stored in `~/.config/backgammon/config.json`. Environment variables fill blanks only. LLM answers are validated against legal plays; illegal replies fall back to Intermediate.

## Difficulty benchmark

```sh
task benchdiff -- 32
# or: go run ./cmd/benchdiff 32
```

Plays seeded self-play series with alternating colors and prints win rates with Wilson confidence intervals.

## Build

[Task](https://taskfile.dev/) drives the Linux build:

```sh
task test
task build    # bin/backgammon
task run
```

Ebitengine needs a C compiler, OpenGL, X11, and ALSA headers.

Fedora:

```sh
sudo dnf install golang gcc pkgconf-pkg-config mesa-libGL-devel libX11-devel libXrandr-devel libXinerama-devel libXcursor-devel libXi-devel libXxf86vm-devel alsa-lib-devel
```

Debian and Ubuntu:

```sh
sudo apt install golang gcc pkg-config libgl1-mesa-dev libx11-dev libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev libxxf86vm-dev libasound2-dev
```

The UI font is Liberation Sans, under the SIL Open Font License (`assets/fonts/OFL.txt`).

## Releases

Pushing a version tag publishes Linux, Windows, and macOS binaries:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions runs tests, builds `open-backgammon-<tag>-<os>-<arch>` (with `.exe` on Windows), and attaches the files plus `SHA256SUMS.txt` to the GitHub Release.

## Layout

- `cmd/backgammon` — window entry point
- `internal/game` — board, dice, and legal plays
- `internal/bot` — easy, intermediate, hard, and OpenRouter opponents
- `internal/ui` — menu, settings, and board
- `TODO.md` — MCP server, Ollama, and multi-OS follow-up
