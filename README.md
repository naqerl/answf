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

Defaults:
- `-fetch`: `https://google.com/search?q=helloworld` (if omitted)
- `--ws-endpoint`: `wss://browserless.aishift.co`
- `--timeout-ms`: `30000`
- `--fallback-textise`: `true`
- `--textise-base-url`: `https://r.jina.ai/http://`
- `--cache-dir`: `~/.cache/answf`
- `--no-cache`: `false`
- `-md`: disabled (prints HTML by default)

Optional endpoint override:

```bash
BROWSERLESS_WS_ENDPOINT=wss://browserless.aishift.co answf -fetch google.com/search?q=helloworld
```

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
- `-searx-url`: `https://searx.aishift.co` (or `SEARX_URL`)
- `--top`: `0` (all results)
- `-v` / `--verbose`: `false` (hide engine metadata)

Search examples:

```bash
answf -s "systemd sandboxing" --top 5
answf -s "systemd sandboxing" -v
answf -s "golang context logger" --no-cache
```
