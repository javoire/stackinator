# Decision Log

Architectural and design decisions for Stackinator.

## 2026-03-06 — Add merged-parent detection to `stack status`

**Decision**: Add merged-parent detection to `stack status`.

**Context**: `stack status` reported "perfectly synced" even when a parent branch's PR had been merged. `stack sync` already had this detection logic (checking PR state + git ancestor fallback) but `status` did not, so users had no warning until they ran sync.

**Resolution**: Mirror `sync.go`'s merged-parent detection (PR state check + git ancestor fallback) in `status.go:detectSyncIssues()`. This lets `status` report when a parent branch has been merged and the child needs re-parenting.
