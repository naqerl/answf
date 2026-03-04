# Problem

`answf` currently has five functional gaps that block reliable use in real-world web retrieval:

1. Fetching fails outright on bot-protected pages (Anubis/go-away/Cloudflare) because rendering depends only on Browserless Playwright.
2. There is no fallback path after a primary fetch failure, so users get errors instead of content.
3. Search output always prints `engine: ...`, which adds noise for normal CLI use.
4. Search results are emitted in provider order with no ranking and no `--top` trimming.
5. No local caching exists, causing repeated network requests and slower repeated workflows.

# Solution

Implement a deterministic fetch/search pipeline in `main.go` with three additions: fallback retrieval, ranked/filtered output, and filesystem caching.

Chosen implementation:

1. Add explicit CLI options to `config`/`parseFlags` in `main.go`:
- `--fallback-textise` (default `true`) and `--textise-base-url` (default `https://r.jina.ai/http://`) for fallback fetch
- `-v` / `--verbose` to show optional metadata (search engine)
- `--top` (default `0`, meaning all)
- `--cache-dir` (default `~/.cache/answf`), `--no-cache`

2. Refactor fetch path in `main.go:run`/`renderHTML` into a fallback chain:
- Primary: existing Playwright browser fetch (keep current `renderHTML` logic as `fetchWithPlaywright(cfg config) (string, error)`)
- Secondary: textise fallback via `fetchViaTextise(targetURL string, timeout time.Duration, textiseBase string) (string, error)`
- Orchestration: `fetchWithFallback(cfg config) (string, error)` that returns first successful content and preserves primary+fallback errors in a combined message.

3. Add cache read/write wrappers used by both search and fetch:
- New file `cache.go` with `type cacheManager struct { dir string; disabled bool; now func() time.Time }`
- Methods:
  - `Get(key string, ttl time.Duration) (string, bool, error)`
  - `Set(key string, content string) error`
  - `keyForFetch(url string, markdown bool) string`
  - `keyForSearch(query string, searxURL string) string`
- Use TTLs: search `1h`, fetch `24h`.

4. Improve search pipeline in `main.go:runSearch`:
- Keep API call/JSON decode behavior.
- Add ranking stage `rankResults(results []searchResult) []searchResult` with deterministic score rules:
  - +40 docs/manual/wiki hosts (`*.docs.*`, `wiki.*`, `readthedocs.io`, `developer.*`)
  - +25 Stack Overflow / Stack Exchange
  - +10 GitHub repo/docs
  - -20 obvious low-signal hosts (generic content farms list)
  - +5 when title contains query tokens
- Add top trimming stage `applyTop(results []searchResult, top int) []searchResult`.
- Update formatter signature to `formatSearchResults(results []searchResult, showEngine bool) string` and only print engine when verbose.

5. Add tests for deterministic behavior:
- New `main_test.go` for URL normalization, ranking order, top trimming, engine visibility formatting.
- New `cache_test.go` for TTL hit/miss, key stability, and no-cache bypass.

6. Update docs in `README.md` with new flags and examples for fallback, verbosity, top-N, and cache controls.

# Tasks

## Phase 1 - CLI and Config Surface
- [ ] Extend `config` in `main.go:31` with:
  - [ ] `FallbackTextise bool`
  - [ ] `TextiseBaseURL string`
  - [ ] `Verbose bool`
  - [ ] `Top int`
  - [ ] `CacheDir string`
  - [ ] `NoCache bool`
- [ ] Update `parseFlags` in `main.go:41`:
  - [ ] Register `--fallback-textise`, `--textise-base-url`, `-v`, `--verbose`, `--top`, `--cache-dir`, `--no-cache`
  - [ ] Normalize and validate `--top >= 0`
  - [ ] Expand `~` in cache dir and default to `$HOME/.cache/answf`

## Phase 2 - Fetch Fallback Chain
- [ ] Split current `renderHTML` (`main.go:92`) into:
  - [ ] `fetchWithPlaywright(cfg config) (string, error)` containing existing browser logic
  - [ ] `fetchViaTextise(targetURL string, timeout time.Duration, textiseBase string) (string, error)` using `net/http`
  - [ ] `fetchWithFallback(cfg config) (string, error)` for primary->fallback orchestration
- [ ] Route `run(cfg)` (`main.go:84`) through `fetchWithFallback` for non-search requests
- [ ] Preserve markdown conversion behavior for both primary and fallback content paths
- [ ] Return combined actionable error when both methods fail (include both method error contexts)

## Phase 3 - Caching Layer
- [ ] Create `cache.go` with `cacheManager` and file-backed cache implementation
- [ ] Use SHA-256 based keys for fetch/search keys and store as `<cache-dir>/<key>.txt`
- [ ] Add metadata file or mtime-based TTL check for:
  - [ ] Search cache TTL `1h`
  - [ ] Fetch cache TTL `24h`
- [ ] Wire cache reads/writes:
  - [ ] Fetch flow (`fetchWithFallback`) checks cache before network and stores successful content
  - [ ] Search flow (`runSearch`) checks cache before HTTP request and stores formatted/raw results
- [ ] Honor `--no-cache` by bypassing all cache reads/writes

## Phase 4 - Search Ranking and Output Controls
- [ ] Add deterministic ranking helpers in `main.go` (or `search.go` if split):
  - [ ] `scoreSearchResult(r searchResult, query string) int`
  - [ ] `rankResults(results []searchResult, query string) []searchResult`
- [ ] Update `runSearch` (`main.go:179`) to apply ranking then `applyTop`
- [ ] Change `formatSearchResults` signature (`main.go:224`) to accept `showEngine bool`
- [ ] Print `engine: ...` only when `cfg.Verbose` is true
- [ ] Ensure `--top N` returns first `N` ranked results; `0` keeps all

## Phase 5 - Tests
- [ ] Add `main_test.go` with table-driven tests for:
  - [ ] `normalizeHTTPURL` and `normalizeWSEndpoint`
  - [ ] `formatSearchResults(..., showEngine=false)` hides engine lines
  - [ ] `formatSearchResults(..., showEngine=true)` includes engine lines
  - [ ] `rankResults` orders higher-signal domains above generic domains
  - [ ] `applyTop` behavior for `0`, `1`, and `N > len(results)`
- [ ] Add `cache_test.go` with temp-dir tests for:
  - [ ] cache hit within TTL
  - [ ] cache miss after TTL expiry
  - [ ] deterministic key generation
  - [ ] no-cache mode bypass

## Phase 6 - Documentation and Validation
- [ ] Update `README.md`:
  - [ ] Add new flags section and defaults
  - [ ] Add fallback example for blocked URLs
  - [ ] Add search examples: `--top`, `-v`, cache controls
- [ ] Run validation commands:
  - [ ] `go test ./...`
  - [ ] Manual smoke checks:
    - [ ] `answf -fetch <blocked-url> --fallback-textise -md`
    - [ ] `answf -s "systemd sandboxing" --top 5`
    - [ ] `answf -s "systemd sandboxing" -v`
