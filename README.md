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
answf fetch google.com/search?q=helloworld
```

Config file lookup order:
1. `--config /path/to/config.yml`
2. `$XDG_CONFIG/answf/config.yml` (also supports `$XDG_CONFIG_HOME/answf/config.yml`)
3. `$HOME/.config/answf/config.yml`

Example config:

```yaml
fetch:
  playwright_url: "wss://browserless.example/ws"
  timeout_ms: 30000
  fallback_textise: true
  textise_base_url: "https://r.jina.ai"
  format: "html" # html or md
search:
  searx_url: "https://searx.example"
  timeout_ms: 30000
```

CLI flags override config where applicable (`-md`/`-html`, `-top`, `-v`, `--no-cache`).

Markdown output:

```bash
answf fetch https://github.com/browserless/browserless -md
answf fetch https://github.com/browserless/browserless -html
```

Fallback fetch for bot-protected pages:

```bash
answf fetch "https://wiki.archlinux.org/title/Systemd/Sandboxing" -md
```

Search output (plain text results):

```bash
answf search "browserless playwright"
```

Search defaults:
- Endpoint/timeouts are read from config file (`fetch` / `search` sections)
- `-top`: `0` (all results)
- `-v` / `--verbose`: `false` (hide engine metadata)

Search examples:

```bash
answf search "systemd sandboxing" -top 5
answf search "systemd sandboxing" -v
answf fetch "https://example.com" --no-cache
```
