# prr — AI-Powered PR Code Review CLI

[![CI](https://github.com/dotbrains/prr/actions/workflows/ci.yml/badge.svg)](https://github.com/dotbrains/prr/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Run AI-powered code reviews on GitHub pull requests. Outputs structured, human-readable markdown comments for easy copy-paste into GitHub.

## Quick Start

```sh
# Install
go install github.com/dotbrains/prr@latest

# Review the current branch's PR
prr

# Review a specific PR
prr 17509

# Review with a specific agent
prr 17509 --agent gpt

# Review with all configured agents
prr 17509 --all
```

## How It Works

1. `prr` resolves the PR number (from an argument or auto-detects from the current branch via `gh`).
2. Fetches the PR diff and metadata using the GitHub CLI.
3. Sends the diff to an AI agent (Claude by default).
4. Writes structured review comments to `reviews/pr-<number>-<timestamp>/`.

Output is organized as one markdown file per reviewed source file, designed for direct copy-paste into GitHub's PR review interface.

## Installation

### Via `go install`

```sh
go install github.com/dotbrains/prr@latest
```

### Via Homebrew

```sh
brew tap dotbrains/tap
brew install prr
```

### Via GitHub Release

```sh
gh release download --repo dotbrains/prr --pattern 'prr_darwin_arm64.tar.gz' --dir /tmp
tar -xzf /tmp/prr_darwin_arm64.tar.gz -C /usr/local/bin
```

### From source

```sh
git clone https://github.com/dotbrains/prr.git
cd prr
make install
```

## Configuration

```sh
# Create default config
prr config init

# Set your API key
export ANTHROPIC_API_KEY=sk-...

# Check agent status
prr agents
```

Config lives at `~/.config/prr/config.yaml`. See [SPEC.md](SPEC.md) for the full config format.

## Commands

| Command | Description |
|---|---|
| `prr [PR_NUMBER]` | Run AI code review on a PR |
| `prr agents` | List configured agents and their status |
| `prr config init` | Create default config file |
| `prr history` | List past reviews |
| `prr clean` | Remove old review output |

## Output

```
reviews/
  pr-17509-20250311-143000/
    summary.md                        # Overall review
    files/
      src-auth-handler-go.md          # Per-file comments
      src-middleware-session-go.md
```

Each file contains comments organized by line number with severity levels (`critical`, `suggestion`, `nit`, `praise`).

## Dependencies

- **[gh](https://cli.github.com/)** — GitHub CLI (required for PR detection and diff fetching)
- **API key** — for your chosen AI provider (e.g. `ANTHROPIC_API_KEY` for Claude)

## License

MIT
