# answf

> >Codex What acronym names for `answf` can you imagine?
>
> 1. Answer Surf
> 2. AI Network Search Workflow
> 3. Adaptive Search and Web Fetch
> 4. Agentic Search Workflow
> 5. Async Navigation, Search, Web Fetch
> 6. Automated Navigation & Scrape Workflow
> 7. Ask, Navigate, Search, Web Fetch
> 8. A New Search Workflow For

Go CLI that uses [`playwright-go`](https://github.com/playwright-community/playwright-go)
to connect to Browserless and print rendered HTML (or Markdown) for a URL.
It also supports SearXNG search queries.

## Install

```bash
go install github.com/naqerl/answf@latest
```

## Run

```bash
answf -fetch google.com/search?q=helloworld
```

Friendly positional mode (no `-fetch`/`-search` needed):

```bash
answf "systemd sandboxing"                              # search
answf "https://wiki.archlinux.org/title/Systemd/Sandboxing" # fetch
```

Config file lookup order:
1. `--config /path/to/config.yml`
2. `$XDG_CONFIG/answf/config.yml` (also supports `$XDG_CONFIG_HOME/answf/config.yml`)
3. `$HOME/.config/answf/config.yml`

Example config:

```yaml
playwright_url: "wss://browserless.example/ws"
searx_url: "https://searx.example"
playwright_timeout_ms: 30000
search_timeout_ms: 30000
fallback_textise: true
textise_base_url: "https://r.jina.ai"
```

CLI flags still override config values.

Markdown output:

```bash
answf -fetch https://github.com/browserless/browserless -md
```

Fallback fetch for bot-protected pages:

```bash
answf -fetch "https://wiki.archlinux.org/title/Systemd/Sandboxing" --fallback-textise -md
```

Search output (plain text results):

```bash
answf -search "browserless playwright"
answf -s "browserless playwright"
```

Search defaults:
- `-searx-url`: from config file (or explicit `--searx-url`)
- `--top`: `0` (all results)
- `-v` / `--verbose`: `false` (hide engine metadata)

Search examples:

```bash
answf -s "systemd sandboxing" --top 5
answf -s "systemd sandboxing" -v
answf -fetch "https://example.com" --no-cache
```
