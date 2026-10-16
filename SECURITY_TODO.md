# Security remediation checklist

**Audit date:** 2026-10-16
**Repository:** `/home/mparvin/Documents/MyGit/Games/BG`
**Application type:** Local desktop Backgammon client (Ebitengine). No HTTP server, no multi-tenant API, no database. Network use is limited to outbound LLM provider calls (OpenRouter / Ollama).

**Trust boundaries**

| Boundary | Trust |
| --- | --- |
| Local player / UI | Fully trusted principal |
| `~/.config/backgammon/config.json` | Sensitive (API key, Ollama URL) |
| OpenRouter HTTPS API | External; responses treated as untrusted |
| Configurable Ollama base URL | Untrusted destination if poisoned or remote |
| Built-in Easy/Intermediate/Hard bots | Offline; no network |

**Scope notes:** No Dockerfiles, Kubernetes manifests, or CI workflows were present. No production services were contacted. This document is a remediation plan only; application code was not modified during the audit.

---

## Finding counts

| Severity | Confirmed | Suspected / requires investigation | Informational / false-positive context |
| --- | ---: | ---: | ---: |
| Critical | 0 | 0 | 0 |
| High | 1 | 0 | 0 |
| Medium | 2 | 1 | 0 |
| Low | 3 | 0 | 0 |
| Informational | 0 | 0 | 4 |

**Secrets in repository:** None detected. `.env.example` has empty placeholders only. `.gitignore` ignores `.env`. No private keys or live API keys were found in tracked files. (Values are not printed here.)

---

## High priority

- [ ] **SEC-001 — Upgrade Go toolchain past known stdlib advisories**
  - **Severity:** High
  - **Confidence:** High
  - **Status:** Confirmed (dependency/toolchain)
  - **CWE:** CWE-1104 (Use of Unmaintained Third Party Components)
  - **Location:** `go.mod` (`go 1.26.3`); runtime `go version` reported `go1.26.3 linux/amd64`
  - **Vulnerable behavior:** `govulncheck ./...` reported **17** symbol-reachable standard-library vulnerabilities in Go **1.26.3**, with fixes available through **1.26.9** (and intermediate patches such as 1.26.4–1.26.6 for some IDs). Client-relevant examples include TLS handshake message limits (`GO-2026-6090`), `net/url` quadratic path resolution (`GO-2026-6218`), certificate/hostname parsing DoS (`GO-2026-5037`), and related `crypto/tls` / `net/http` client paths exercised by `internal/ollama/client.go` and `internal/openrouter/client.go`.
  - **Preconditions:** Application builds/runs against Go 1.26.3 (or any unfixed patch level) and performs outbound HTTPS/HTTP.
  - **Attack scenario:** A malicious or compromised TLS endpoint (for example a hostile Ollama URL, or a MITM on a non-pinned connection) triggers known stdlib DoS or protocol issues during `http.Client.Do`.
  - **Impact:** Process hang/crash or elevated attack surface on outbound TLS/HTTP; several listed IDs are primarily **server** HTTP/2 issues and are lower practical risk for this desktop client, but the shared recommendation is still to move to a fixed toolchain.
  - **Evidence:** `govulncheck ./...` exit code 3; Symbol Results listed GO-2026-6617, 6613, 6612, 6611, 6610, 6608, 6607, 6605, 6603, 6218, 6090, 6088, 5972, 5856, 5039, 5037, 5026 (and additional import/module findings not called by app code).
  - **Remediation:** Upgrade the local Go toolchain (and `go` directive in `go.mod` if required) to **≥ 1.26.9** (or current latest 1.26.x patch), rebuild, and re-run `govulncheck ./...` until client-reachable findings are cleared or accepted with documented risk.
  - **Regression tests:** CI/`Taskfile` target that runs `govulncheck ./...` and fails on new HIGH client-reachable stdlib findings.
  - **Tool status:** Confirmed by `govulncheck` (official vulnerability DB).

---

## Medium priority

- [ ] **SEC-002 — Unvalidated Ollama base URL enables SSRF / local service probing**
  - **Severity:** Medium
  - **Confidence:** High
  - **Status:** Confirmed
  - **CWE:** CWE-918 (Server-Side Request Forgery)
  - **Location:** `internal/ollama/client.go` (`base` ≈ L48–54, `Models` ≈ L64–69, `Complete` ≈ L111–125); UI wiring `internal/ui/app.go` (`llmCompleter` ≈ L365–368, `checkProvider` ≈ L919–929, settings field ≈ L795–798); config `internal/config/config.go` (`OllamaURL` / `OLLAMA_HOST` ≈ L62–67)
  - **Vulnerable behavior:** The Ollama `BaseURL` is taken from user settings / `OLLAMA_HOST` with **no scheme/host allowlist**. Requests are issued with the default `http.Client`, which **follows redirects** (no `CheckRedirect`). Path suffixes `/api/tags` and `/v1/chat/completions` are concatenated onto the attacker-chosen base.
  - **Preconditions:** Ability to set `ollama_url` in Settings (or poison `config.json` / `OLLAMA_HOST`). On a multi-user host, that implies write access to the victim’s config or environment.
  - **Attack scenario:** Point the base URL at an internal service (`http://127.0.0.1:…`, link-local metadata `http://169.254.169.254/`, or an external host that **302-redirects** to those targets). The game process issues authenticated-as-local-user HTTP GETs/POSTs and surfaces status codes/errors in the UI.
  - **Impact:** Local/network service discovery, interaction with loopback admin APIs that trust localhost, or cloud metadata access when the desktop runs on a cloud VM. Impact is reduced vs a multi-tenant server (the operator is usually the same user), but config poisoning or social engineering remains realistic.
  - **Evidence:** Manual review; `rg` confirmed no `CheckRedirect` / `InsecureSkipVerify` anywhere. Settings UI accepts arbitrary URL text. `gosec` did not specifically flag SSRF; finding is from code review.
  - **Remediation:** Restrict schemes (`http`/`https` only), optionally default-allow only loopback / explicit user confirmation for non-loopback hosts, strip `userinfo`, and set `http.Client.CheckRedirect` to refuse redirects to link-local, loopback (when origin was not loopback), or metadata ranges. Consider blocking literal IPs in cloud metadata ranges.
  - **Regression tests:** Unit tests that reject `file:`, credentialed URLs, and redirect-to-169.254.169.254; allow `http://127.0.0.1:11434`.
  - **Tool status:** Manual confirmation; not reported by gosec as SSRF.

- [ ] **SEC-003 — Cleartext remote Ollama traffic and plaintext API key on disk**
  - **Severity:** Medium (remote HTTP) / Low–Medium (disk key)
  - **Confidence:** High
  - **Status:** Confirmed
  - **CWE:** CWE-319 (Cleartext Transmission of Sensitive Information); CWE-312 (Cleartext Storage of Sensitive Information)
  - **Location:**
    - Disk: `internal/config/config.go` `Save` ≈ L78–92 (`json.MarshalIndent` of `APIKey` as `api_key`, file mode `0o600`); directory `MkdirAll(..., 0o755)` ≈ L85
    - Wire: `internal/ollama/client.go` default `http://localhost:11434` (L16); user may set any `http://` remote host
    - Display: `internal/ui/app.go` `drawField(..., secret bool)` ≈ L816–824 (key visible while focused)
  - **Vulnerable behavior:** OpenRouter API keys are written in plaintext JSON under the user config dir. Config **directory** is created as `0755` while the file is `0600`. Ollama requests (including board state in chat messages from `internal/bot/ai.go` `prompt`) use whatever scheme the URL specifies; a remote `http://` Ollama host sends game prompts in cleartext.
  - **Preconditions:** User saves an API key; or configures a non-localhost HTTP Ollama URL; or another local account can list `0755` directories (file content still requires `0600` read).
  - **Attack scenario:** Local malware or another user with compromised account reads `config.json`. Network attacker on the path to a remote HTTP Ollama instance observes prompts/API traffic.
  - **Impact:** OpenRouter key theft (billing / quota abuse); confidentiality loss of game state sent to remote Ollama.
  - **Evidence:** Code review; `gosec` **G117** on `config.go:88` (secret marshaled to JSON); `gosec` **G301** on `config.go:85` and `stats/history.go:56` (directory `0755`).
  - **Remediation:** Create config dirs as `0700`. Prefer OS keyring / secret service for `APIKey`, or document that env-only keys should not be persisted. Warn or block non-TLS remote Ollama URLs. Keep masking of the key when the field is unfocused (already implemented).
  - **Regression tests:** Assert config directory mode `0700` after `Save`; optional test that remote `http://` Ollama URLs are rejected or warned.
  - **Tool status:** Confirmed by review + gosec G117/G301.

- [ ] **SEC-004 — `config.Save` may re-persist an environment API key the UI cleared** *(suspected product footgun / privacy)*
  - **Severity:** Medium (privacy) — treat as product defect with security impact
  - **Confidence:** Medium
  - **Status:** Suspected / requires product decision
  - **CWE:** CWE-212 (Improper Removal of Sensitive Information Before Storage)
  - **Location:** `internal/config/config.go` `Save` → `applyEnv` (L79–80, L48–51); UI `saveSettings` ≈ L943–960
  - **Vulnerable behavior:** If the user clears the API key field and clicks Save while `OPENROUTER_API_KEY` is set in the environment, `applyEnv` refills `APIKey` before write, so the key may be written to `config.json` unexpectedly.
  - **Preconditions:** Env var set; user intends to remove the persisted key.
  - **Impact:** Unexpected long-term persistence of a secret the user believed they removed from disk.
  - **Evidence:** Code path review only (no end-to-end UI test executed for this case).
  - **Remediation:** Do not call `applyEnv` on Save for secrets already represented in the form; or distinguish “unset” vs “explicitly cleared”; document that env overrides apply only on Load.
  - **Regression tests:** Save with empty `APIKey` and env set must not write the env value unless explicitly requested.
  - **Tool status:** Manual; needs UX confirmation.

---

## Low priority

- [ ] **SEC-005 — Unbounded LLM `reason` / response text placed into UI status**
  - **Severity:** Low
  - **Confidence:** High
  - **Status:** Confirmed
  - **CWE:** CWE-770 (Allocation of Resources Without Limits or Throttling) — soft; also content injection into UI status
  - **Location:** `internal/bot/ai.go` L47–57 (`a.Note = d.Reason`; retry appends full `text`); `internal/ui/app.go` `startBot`/`pollBot` ≈ L465–484 (`a.status = res.note`)
  - **Vulnerable behavior:** Model `reason` and retry bodies are not length-capped beyond the HTTP body read limit (`1<<20` in clients). A cooperative or malicious model can return a very large `reason`, which is copied into `App.status` and drawn each frame.
  - **Preconditions:** AI opponent enabled with a reachable Completer that returns oversized JSON/`reason`.
  - **Impact:** UI slowdown or memory pressure; not board corruption (play index is bounds-checked against `LegalPlays`).
  - **Evidence:** Code review; body limit exists at L78/L130 of HTTP clients, but no additional cap on `Decision.Reason` before UI use.
  - **Remediation:** Truncate `Reason` (e.g. 200 runes) before storing `Note`/status; drop oversized assistant text on retries.
  - **Regression tests:** Completer returning a multi-megabyte `reason` must not grow status beyond the cap; illegal `play_index` still falls back safely (existing tests cover index validation).
  - **Tool status:** Manual.

- [ ] **SEC-006 — Provider check / Ollama errors may echo attacker-influenced strings into the UI**
  - **Severity:** Low
  - **Confidence:** Medium
  - **Status:** Confirmed (limited impact)
  - **CWE:** CWE-209 (Generation of Error Message Containing Sensitive Information)
  - **Location:** `internal/ui/app.go` `checkProvider` L928–937 (`a.statusNote = err.Error()`); `internal/ollama/client.go` error wrapping L71, L107, L142
  - **Vulnerable behavior:** Raw `error.Error()` strings (including model names and some transport errors) are shown in Settings. OpenRouter Check intentionally avoids echoing the API key (verified by `TestCheckRejectsBadKey`). URLs with embedded credentials could still appear in dial errors from the standard library.
  - **Impact:** Minor information disclosure on screen; not remote exfiltration.
  - **Remediation:** Map errors to fixed user-facing messages; never include raw URL userinfo.
  - **Regression tests:** Extend provider tests so dial errors with `http://user:pass@host` never surface the password in `statusNote`.
  - **Tool status:** Manual + existing openrouter key non-leak test.

- [ ] **SEC-007 — Document and pin outbound HTTP clients (timeouts exist; redirect policy missing)**
  - **Severity:** Low
  - **Confidence:** High
  - **Status:** Confirmed (hardening)
  - **CWE:** CWE-918 (related); CWE-400
  - **Location:** `internal/ollama/client.go` L56–60; `internal/openrouter/client.go` L69–71, L113–115
  - **Vulnerable behavior:** Clients set timeouts (good) and cap response bodies at 1 MiB (good) but inherit default redirect following and default TLS config (no certificate pinning). Acceptable for OpenRouter’s fixed HTTPS host; weaker when Ollama URL is arbitrary (see SEC-002).
  - **Remediation:** Shared `http.Client` helper with explicit `CheckRedirect`, optional pinned OpenRouter host allowlist for `Endpoint`/`ModelsURL` outside tests.
  - **Regression tests:** Redirect count / destination policy unit tests.
  - **Tool status:** Manual.

---

## Informational / rejected scanner noise

These were reviewed and **should not be treated as product vulnerabilities** without new evidence:

- [ ] **INFO-001 — gosec G404 (`math/rand`) on dice and bot RNG**
  - **Severity:** Informational (scanner HIGH is **not applicable**)
  - **Status:** False positive for this threat model
  - **Location:** `internal/ui/app.go:139`, `internal/bot/search.go:20`, `internal/bot/selfplay.go:81`, `cmd/benchdiff/main.go:19–21`
  - **Evidence:** RNG is used for game dice and AI move sampling, not for secrets, session tokens, or auth. Predictable dice do not yield remote compromise in a local single-player/hotseat game.
  - **Remediation:** No change required for security; optionally `#nosec G404` with comment, or silence in gosec config for these paths.

- [ ] **INFO-002 — gosec G304 “file inclusion via variable” on config/history paths**
  - **Severity:** Informational
  - **Status:** False positive
  - **Location:** `internal/config/config.go:36`, `internal/stats/history.go:41`
  - **Evidence:** Paths are derived from `os.UserConfigDir()` + fixed filenames (`backgammon/config.json`, `history.json`), not from raw attacker path input in the UI.
  - **Remediation:** None; keep using fixed basenames under the user config directory.

- [ ] **INFO-003 — No listening HTTP server / no authn-authz surface**
  - **Status:** Residual risk note
  - **Evidence:** `rg` found no `Listen`/`http.Server` in application packages (only `httptest` in tests). Classic web issues (IDOR, JWT, SQL injection, session fixation) **do not apply** to the current architecture.

- [ ] **INFO-004 — LLM cannot mutate the board directly**
  - **Status:** Positive control
  - **Location:** `internal/bot/ai.go` `Choose`
  - **Evidence:** Responses are parsed for `play_index` only; out-of-range or malformed answers retry then fall back to Intermediate. Engine applies `LegalPlays` / roll steps in the UI. This is a strength, not a finding.

---

## Areas not fully verified

| Area | Reason |
| --- | --- |
| Live OpenRouter / live Ollama | Intentionally not contacted; only `httptest` mocks |
| Container / CI supply chain | No Docker/CI files in repo |
| Dependency CVEs beyond stdlib call graph | `govulncheck` noted 6 import + 3 module vulns **not called** by app code; not itemized as confirmed exploitable |
| `staticcheck` | Installed binary built with Go 1.23; incompatible with module `go 1.26.3` — results unreliable |
| Interactive UI secret masking under focus | Code review only; no GUI automation |
| Multi-user Linux ACL / SELinux edge cases | Not exercised |

---

## Tools executed

| Tool | Result |
| --- | --- |
| `go test ./... -count=1 -short` | **Pass** (exit 0) |
| `go vet ./...` | **Pass** (exit 0) |
| `govulncheck ./...` | **Fail** (exit 3): 17 stdlib symbol-reachable vulns on Go 1.26.3; additional uncalled import/module hits |
| `gosec ./...` | **Fail** (exit 1): 13 issues — mostly G404 (RNG), G304 (paths), G117 (API key JSON), G301 (dir 0755) |
| `staticcheck ./...` | **Unavailable / unreliable** (tool built with Go 1.23 vs module 1.26.3) |
| Secret pattern scan (`rg` for private keys / live key prefixes) | **No live secrets found** |
| Manual review | SSRF/redirect, secret storage, LLM status path, architecture |

---

## Suggested remediation order

1. SEC-001 (Go upgrade)
2. SEC-002 (Ollama URL / redirect policy)
3. SEC-003 (config dir mode + key/TLS hygiene)
4. SEC-004 (Save vs env key behavior)
5. SEC-005 / SEC-006 / SEC-007 (LLM status caps, error mapping, shared HTTP client)

Do not implement fixes in the audit phase; track completion by checking boxes above in a follow-up change.
