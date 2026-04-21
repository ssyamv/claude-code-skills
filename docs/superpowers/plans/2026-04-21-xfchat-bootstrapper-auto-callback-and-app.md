# Xfchat Bootstrapper Auto Callback And App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automatically recover from callback port conflicts and missing `lark_cli` apps during Xfchat bootstrap.

**Architecture:** Keep callback port selection inside `internal/orchestrator`, make platform setup accept the runtime callback URL, and make `internal/platformapi.Client.CreateApp` search first then create if missing. The actual callback URL becomes the value used for redirect setup and OAuth URL generation.

**Tech Stack:** Go standard library HTTP/net packages, existing internal platform API client, existing Go unit tests.

---

### Task 1: Callback Port Fallback

**Files:**
- Modify: `internal/orchestrator/callback.go`
- Test: `internal/orchestrator/callback_test.go`

- [ ] Write a failing test that occupies `127.0.0.1:8080`, calls the new preferred callback starter, and asserts the returned URL is not `:8080`.
- [ ] Run `go test ./internal/orchestrator -run TestStartCallbackServerFallsBackToEphemeralWhenDefaultPortUnavailable -count=1` and confirm it fails because the fallback function is missing or still returns the bind error.
- [ ] Implement `StartCallbackServerWithFallback(preferredAddress string)` and make `StartCallbackServer` call it with `defaultCallbackAddress`.
- [ ] Run the same test and confirm it passes.

### Task 2: Runtime Callback URL Data Flow

**Files:**
- Modify: `internal/orchestrator/oauth.go`
- Modify: `internal/orchestrator/orchestrator.go`
- Test: `internal/orchestrator/orchestrator_test.go`

- [ ] Write a failing orchestrator test where the callback waiter URL is `http://127.0.0.1:18081/callback` and assert platform setup receives that runtime URL when ensuring redirects and building `AuthURL`.
- [ ] Run `go test ./internal/orchestrator -run TestRunUsesRuntimeCallbackURLForPlatformSetupAndOAuth -count=1` and confirm it fails because setup still uses `cfg.CallbackURL`.
- [ ] Change the platform setup runner contract so stateful setup can receive the runtime callback URL.
- [ ] Start the callback server before platform setup when setup is required, then pass `waiter.URL()` into platform setup and OAuth.
- [ ] Generate `AuthURL` with the runtime callback URL.
- [ ] Run the focused orchestrator tests and confirm they pass.

### Task 3: Create Missing App

**Files:**
- Modify: `internal/platformapi/client.go`
- Modify: `internal/platformapi/types.go`
- Test: `internal/platformapi/client_test.go`

- [ ] Write a failing test where app list returns no `lark_cli`, then the client sends a create request and parses the created app id.
- [ ] Run `go test ./internal/platformapi -run TestClientCreateAppCreatesWhenExistingAppMissing -count=1` and confirm it fails because the client currently returns `existing app "lark_cli" not found`.
- [ ] Add the minimal create-app request after the list lookup misses.
- [ ] Run the focused platform API test and confirm it passes.

### Task 4: Full Verification

**Files:**
- Existing Go packages

- [ ] Run `go test ./internal/orchestrator ./internal/platformapi ./internal/config ./internal/preflight`.
- [ ] Run `go test ./...`.
- [ ] Inspect `git diff --stat` and `git diff` to confirm only intended files changed.
