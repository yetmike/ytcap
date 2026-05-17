# ytcap

A distraction-free YouTube TUI with AI-powered summaries and Q&A, inspired by [k9s](https://k9scli.io/).

Browse channels, search videos, read transcripts, and chat with an AI about any video — all from your terminal, no algorithm pushing you around.

## Features

- **Search** YouTube and browse channels without opening a browser
- **AI summaries** of videos in multiple lengths (short → xl)
- **Transcript viewing** with fast scrolling and search
- **Q&A chat** grounded in the video transcript
- **Save notes** as markdown to your local knowledge base
- **Offline-friendly cache** for transcripts and summaries
- **Customizable skins and keybindings** via YAML
- Non-interactive CLI mode — pipe summaries and transcripts into other tools

## Requirements

`ytcap` shells out to two external tools:

- [`yt-dlp`](https://github.com/yt-dlp/yt-dlp) — YouTube data + transcripts
- [`summarize`](https://github.com/steipete/summarize) — AI summary backend (wraps Claude / Gemini)

Install them first:

```sh
# yt-dlp
brew install yt-dlp       # macOS
pip install yt-dlp        # any platform

# summarize
npm i -g @steipete/summarize
```

## Install

### Homebrew

```sh
brew install yetmike/tap/ytcap
```

### Go

```sh
go install github.com/yetmike/ytcap@latest
```

### Pre-built binaries

Grab a release from the [releases page](https://github.com/yetmike/ytcap/releases).

### From source

```sh
git clone https://github.com/yetmike/ytcap
cd ytcap
make build
```

## Usage

### Interactive TUI

```sh
ytcap                          # open the TUI
ytcap "rust async"             # open straight into search results
ytcap --channel @fireship      # jump to a channel
ytcap --video <url>            # open a specific video
```

### Non-interactive CLI

```sh
ytcap summary <url>            # print AI summary to stdout
ytcap transcript <url>         # print transcript to stdout
```

Pipe into anything:

```sh
ytcap summary https://youtu.be/... | glow -
ytcap transcript https://youtu.be/... | wc -w
```

### Config and cache

```sh
ytcap config show              # print current config
ytcap config edit              # open config in $EDITOR
ytcap config reset             # restore defaults
ytcap cache clear              # clear all cached data
ytcap cache clear <video-id>   # clear one video
```

## Configuration

Config lives at `~/.ytcap/config.yaml` and is created with inline comments on first run. Key sections:

```yaml
summarize:
  length: "long"         # short | medium | long | xl
  language: "auto"       # auto | en | de | ...
  cli: "auto"            # auto | claude | gemini

ytdlp:
  search_limit: 20

cache:
  enabled: true
  ttl_days: 7

storage:
  dir: "~/ytcap-notes"

ui:
  skin: "default"        # drop custom skins in ~/.ytcap/skins/<name>.yaml
  mouse: false
```

Cached transcripts and summaries live in `~/.ytcap/cache/`. Notes saved from the TUI (`S` / `C`) go to `~/ytcap-notes/`.

Set `YTCAP_LOG=debug` for verbose logging to stderr in addition to `~/.ytcap/ytcap.log`.

## Keybindings

Defaults (rebindable in config):

| Key       | Action        |
| --------- | ------------- |
| `?`       | Help          |
| `:`       | Search        |
| `/`       | Filter / chat |
| `Tab`     | Toggle view   |
| `S`       | Save summary  |
| `C`       | Save chat     |
| `Esc`     | Back          |
| `q`       | Quit          |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
