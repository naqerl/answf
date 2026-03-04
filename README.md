# answf

Go `1.26` CLI that uses [`playwright-go`](https://github.com/playwright-community/playwright-go)
to connect to Browserless and print rendered HTML (or Markdown) for a URL.

## Run

```bash
go run . -fetch google.com/search?q=helloworld
```

Defaults:
- `-fetch`: `https://google.com/search?q=helloworld` (if omitted)
- `--ws-endpoint`: `wss://browserless.aishift.co`
- `--timeout-ms`: `30000`
- `-md`: disabled (prints HTML by default)

Optional endpoint override:

```bash
BROWSERLESS_WS_ENDPOINT=wss://browserless.aishift.co go run . -fetch google.com/search?q=helloworld
```

Markdown output:

```bash
go run . -fetch https://github.com/browserless/browserless -md
```
