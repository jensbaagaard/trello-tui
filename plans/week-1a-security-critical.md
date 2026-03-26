# Week 1a: Security — Critical & High Fixes

Priority: CRITICAL / HIGH findings from the SecOps review.

---

## 1. Sanitize attachment filenames and add temp file cleanup

**Severity**: CRITICAL
**Files**: `internal/trello/client.go:207-244`, `internal/tui/card.go:508-518`

### Problem

The `DownloadAttachment` function uses the attachment name from the Trello API
directly in `os.CreateTemp()` to derive the file extension. A crafted attachment
with a `.app` or `.scpt` extension would be opened by `exec.Command("open", path)`,
potentially executing arbitrary code. Additionally, temp files are never cleaned up.

### Changes

**client.go — `DownloadAttachment()`**:

1. Allowlist safe file extensions (images, PDFs, text, etc.). Reject or rename
   unknown extensions to `.bin`.
2. Sanitize `att.Name` — strip path separators, null bytes, and non-printable
   characters before extracting the extension.
3. Add `os.Remove(tmp.Name())` on error paths so partial downloads don't linger.
4. Return a cleanup function or document that the caller is responsible for cleanup.

```go
// Proposed extension allowlist
var safeExtensions = map[string]bool{
    ".txt": true, ".pdf": true, ".png": true, ".jpg": true,
    ".jpeg": true, ".gif": true, ".svg": true, ".webp": true,
    ".csv": true, ".json": true, ".xml": true, ".html": true,
    ".md": true, ".zip": true, ".doc": true, ".docx": true,
    ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
}
```

**card.go — `openAttachment()`**:

1. After `exec.Command("open", path).Start()` completes, schedule cleanup:
   use a goroutine with a delay or `defer` the removal in the calling context.
2. Consider using `xdg-open` on Linux / `open` on macOS with platform detection.

### Tests to add

- `TestDownloadAttachment_SanitizesExtension` — malicious `.app` becomes `.bin`
- `TestDownloadAttachment_CleansUpOnError` — verify temp file removed on write failure
- `TestDownloadAttachment_AllowsSafeExtensions` — `.pdf`, `.png` pass through

---

## 2. Strip credentials from error messages

**Severity**: HIGH
**Files**: `internal/trello/client.go:36-88`

### Problem

API errors include the full response body via `fmt.Errorf("API error %d: %s", ...)`.
If the Trello API echoes back the request URL or token in error responses, credentials
leak into TUI status messages and potential log output.

### Changes

**client.go — `get()` and `request()` methods**:

1. Do not pass raw API response body to error messages shown in the TUI.
2. Create user-friendly error messages based on HTTP status codes:

```go
func apiError(statusCode int, body []byte) error {
    switch statusCode {
    case 401:
        return fmt.Errorf("authentication failed — check your API key and token")
    case 403:
        return fmt.Errorf("access denied — you may not have permission for this resource")
    case 404:
        return fmt.Errorf("resource not found")
    case 429:
        return fmt.Errorf("rate limited — please wait and try again")
    default:
        // Truncate body, strip anything that looks like a token
        msg := string(body)
        if len(msg) > 200 {
            msg = msg[:200] + "..."
        }
        return fmt.Errorf("API error %d: %s", statusCode, msg)
    }
}
```

3. Log the full error details to stderr only when a `--debug` flag is set (future work).

### Tests to add

- `TestApiError_401_NoCredentialsInMessage`
- `TestApiError_TruncatesLongBody`

---

## 3. Fix `io.ReadAll` error swallowing

**Severity**: BUG
**Files**: `internal/trello/client.go:52, 80, 230`

### Problem

```go
body, _ := io.ReadAll(resp.Body)  // error silently discarded
```

Three locations discard the error from `io.ReadAll`. If reading fails (e.g., connection
reset mid-response), the error context is lost and a misleading empty-body error is shown.

### Changes

Handle the error at all three locations:

```go
body, readErr := io.ReadAll(resp.Body)
if readErr != nil {
    return fmt.Errorf("API error %d (could not read response: %w)", resp.StatusCode, readErr)
}
return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
```

### Tests to add

- `TestGet_ReadBodyError` — mock a reader that returns an error
- `TestRequest_ReadBodyError` — same for `request()`

---

## Checklist

- [x] Attachment filename sanitization with extension allowlist
- [x] Temp file cleanup on error paths
- [x] User-friendly API error messages (no credential leakage)
- [x] Handle `io.ReadAll` errors at all three call sites
- [x] Tests for all changes
- [x] Run `go vet ./...` and `go test ./...` pass
