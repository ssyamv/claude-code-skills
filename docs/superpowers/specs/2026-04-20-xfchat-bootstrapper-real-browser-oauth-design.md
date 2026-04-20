# Xfchat Bootstrapper Real Browser And OAuth Design

**Date:** 2026-04-20
**Status:** Draft approved for spec write
**Owner:** Codex

## Summary

Implement the real browser automation and OAuth behavior inside `xfchat-bootstrapper` so the product can perform the actual bootstrap flow instead of stopping at internal unimplemented runner errors.

This design builds on the standalone-runtime refactor and defines the first real execution path for:

- browser-based open-platform automation
- localhost OAuth callback handling
- internal runtime state updates
- internal validation based on bootstrapper-owned state/config

The confirmed product constraints are:

- reuse the employee's already logged-in local Chrome or Edge profile by default
- continue using `http://localhost:8080/callback`
- keep only bootstrapper-owned local state/config
- do not export a parallel `lark-cli` config as part of the normal path

## Goals

- Implement real browser automation against `open.xfchat.iflytek.com`
- Reuse an already logged-in Chrome or Edge local profile by default
- Complete the OAuth round trip through a bootstrapper-owned localhost callback listener
- Store only bootstrapper-owned local state/config
- Advance runtime phases based on real internal work instead of unimplemented stubs

## Non-Goals

- Reintroducing `lark-cli` as a normal dependency
- Building a general browser automation framework for arbitrary sites
- Supporting Linux in this stage
- Implementing package-manager distribution
- Solving every future UI change of the open platform in this spec

## Product Decisions

- Default browser strategy is to reuse the logged-in local Chrome/Edge user profile
- OAuth callback remains `http://localhost:8080/callback`
- Bootstrapper owns the callback server
- Bootstrapper owns local state/config entirely
- There is no required `lark-cli` config export path in the normal flow

## Recommended Architecture

Keep the current standalone runtime structure, but replace the browser and OAuth skeletons with real implementations.

The runtime execution stack becomes:

1. `main`
2. `orchestrator`
3. `preflight`
4. `browser profile resolver`
5. `open-platform automation runner`
6. `oauth callback server`
7. `internal validation`
8. `state + diagnostics`

## Browser Session Design

### Default Mode

Use the employee's already logged-in Chrome or Edge profile by default.

This means the browser automation should:

- locate a supported installed browser
- reuse its local user-data directory
- launch automation in a way that can leverage the existing authenticated session

The first release of the real browser path only needs to support the known target:

- Chrome
- Edge

If neither is available, fail with a clear browser prerequisite error.

### Browser Selection

Preferred order:

1. local Chrome profile
2. local Edge profile

The runtime should not silently fall back to a fresh ephemeral browser profile in this stage, because the user explicitly wants the logged-in-session experience.

## Open Platform Automation Flow

The open-platform runner must automate:

1. open `https://open.xfchat.iflytek.com/app`
2. create a new app
3. navigate to base info
4. read `App ID`
5. reveal and read `App Secret`
6. navigate to safe settings
7. ensure `http://localhost:8080/callback` exists
8. navigate to auth settings
9. apply required scopes
10. publish a version

The browser runner may still use selectors and step sequencing from the existing `internal/browser` package, but it must now execute those steps for real.

## OAuth Design

### Callback Listener

The bootstrapper must own a local callback listener on:

```text
http://localhost:8080/callback
```

Responsibilities:

- start the listener before opening or continuing the authorization step
- receive the callback request
- capture the success/failure payload needed to mark runtime progress
- shut down cleanly after completion or timeout

### Authorization Flow

After the platform runner has configured the app and published it, the runtime should:

- reach the authorization page in the browser
- wait for user/browser completion if explicit confirmation is required
- capture the callback on `localhost:8080`
- move runtime state into `validate`

This must not delegate to `lark-cli auth login`.

## Local State And Config Ownership

Bootstrapper-owned state remains the only required local persisted state.

Persist at minimum:

- current phase
- `App ID`
- `App URL`
- auth success marker
- timestamps or status metadata useful for resume

There is no normal-path requirement to emit a separate `lark-cli` config file.

## Runtime Validation

Validation should become bootstrapper-specific instead of shell-based.

The first real internal validation should confirm:

- platform setup produced a persisted app identity
- OAuth callback completed successfully
- runtime state advanced to the correct phase

It does not need to validate every downstream business capability yet. The main purpose is to prove that bootstrapper-owned setup completed.

## Preflight Updates

Existing preflight checks must remain, but the real runtime needs a few additional guarantees:

- supported browser exists locally
- browser user-data/profile path is discoverable
- port `8080` is free
- writable local state/config location exists

If these fail, the runtime should stop before browser automation starts.

## Diagnostics Requirements

The real browser/oAuth implementation must preserve or improve diagnostics.

Capture when relevant:

- current phase
- last browser URL
- screenshot before failure
- HTML snapshot before failure
- callback server status
- state file contents with secrets redacted

Diagnostics remain especially important because browser automation failures are harder to reason about than CLI failures.

## Error Handling

### Browser Errors

Examples:

- no supported browser found
- failed to attach to reusable browser profile
- selector not found
- page structure changed

Behavior:

- stop
- classify as user-actionable or platform-actionable as appropriate
- emit diagnostics

### OAuth Errors

Examples:

- callback server could not bind `localhost:8080`
- callback timed out
- callback returned error

Behavior:

- stop
- record phase and diagnostics
- allow rerun through existing state logic

### Validation Errors

Examples:

- app data missing after platform setup
- callback success not persisted

Behavior:

- stop with an internal validation error
- do not mention missing `lark-cli`

## Testing Strategy

### Unit Tests

Cover:

- browser profile resolution logic
- state transitions around platform setup and OAuth
- callback listener lifecycle
- validation against bootstrapper-owned state

### Integration Tests With Fakes

Cover:

- fake browser runner returns app metadata and advances state
- fake callback server marks auth success
- orchestrator advances `platform_setup -> oauth -> validate`

### Regression Tests

Keep and extend the regression that ensures:

- startup does not fail because an external `lark-cli` binary is missing

Also add regression coverage for:

- `localhost:8080` binding conflicts
- missing browser profile
- validation after successful callback

### Manual Validation

Before release, perform at least:

- clean-machine install of the bootstrapper
- first run with no external `lark-cli`
- observed browser automation against the real open platform
- successful callback to `localhost:8080/callback`
- persisted state confirmed after success

## Rollout Plan

### Step 1

Implement browser profile discovery and attach strategy.

### Step 2

Implement real open-platform browser automation with current selectors.

### Step 3

Implement real OAuth callback server and integrate it into the orchestrator flow.

### Step 4

Replace placeholder internal validation with bootstrapper-owned validation checks.

### Step 5

Validate that a clean-machine installed bootstrapper completes real internal work without requiring `lark-cli`.

## Explicit Product Decisions

- Logged-in local browser profile is the default automation target
- OAuth remains localhost callback based
- Bootstrapper owns callback handling
- Bootstrapper owns local state/config only
- No normal-path `lark-cli` config export

## Implementation Boundary

This spec only covers the real internal runtime behavior. It does not change the one-line installer distribution design, release artifact naming, or GitHub Releases flow.
