# Xfchat Bootstrapper Standalone Runtime Design

**Date:** 2026-04-20
**Status:** Draft approved for spec write
**Owner:** Codex

## Summary

Refactor `xfchat-bootstrapper` so it becomes a true standalone runtime and no longer depends on an external `lark-cli` executable at runtime.

Today the installer can download and launch `xfchat-bootstrapper`, but the program immediately tries to execute `lark-cli bootstrap` or `lark-cli validate` from a computed path such as:

```text
C:\Users\<user>\AppData\Roaming\XfchatLarkCli\bin\lark-cli.exe
```

That path does not exist on a clean machine because the one-line installer only installs `xfchat-bootstrapper`, not `lark-cli`. As a result, the release is installable but not usable.

This spec changes the runtime contract:

- `xfchat-bootstrapper` must execute the bootstrap flow itself
- the main runtime path must not require `lark-cli.exe`
- browser automation, OAuth handling, local config/state updates, and diagnostics remain internal concerns of the bootstrapper

The external `lark-cli` adapter may remain only as a compatibility or future export layer, but it must not be part of the normal startup path.

## Goals

- Remove the runtime dependency on external `lark-cli`
- Make `xfchat-bootstrapper` start and progress on a clean machine after one-line installation
- Keep the installer/release UX unchanged for end users
- Reuse the existing modules where reasonable: `preflight`, `browser`, `state`, `diagnostics`
- Preserve resumable state and diagnostics as first-class runtime features

## Non-Goals

- Replacing the one-line installer distribution model
- Reintroducing `lark-cli` as a required install dependency
- Shipping Linux support
- Solving every browser automation edge case in this spec
- Defining package-manager support

## Root Cause

The current startup path is structurally wrong for the desired product.

At startup:

- `main` builds config and store
- `main` calls `orchestrator.New(...)`
- `orchestrator.New(...)` constructs a `larkcli.Adapter`
- `Validate` and `Execute` invoke external commands instead of internal runtime logic

This means the binary that users install is only an orchestrator shell, not the actual bootstrap tool. On a clean machine, startup fails before meaningful work begins because the external CLI path does not exist.

The bug is therefore not “missing file” in isolation. The bug is that the architecture still assumes another product is installed.

## Product Decision

`xfchat-bootstrapper` becomes the product entrypoint and the runtime implementation.

This means:

- browser automation is owned by `xfchat-bootstrapper`
- OAuth/callback handling is owned by `xfchat-bootstrapper`
- local configuration writes are owned by `xfchat-bootstrapper`
- resumable state is owned by `xfchat-bootstrapper`
- diagnostics are owned by `xfchat-bootstrapper`

## Recommended Architecture

Keep the current package structure where possible, but change the runtime call graph so the orchestrator invokes internal runners instead of external commands.

The runtime should consist of these internal subsystems:

1. `preflight`
2. `platform setup runner`
3. `oauth runner`
4. `state store`
5. `diagnostics`
6. `thin main entrypoint`

## Runtime Phases

The standalone runtime should treat bootstrap as a sequence of internal phases:

1. `preflight`
2. `platform_setup`
3. `oauth`
4. `validate`

These phase names can continue to live in `internal/state` if they are still sufficient. If more granularity is needed later, additional phases may be introduced, but the first correction should keep the phase model as small as possible.

## Component Design

### Thin Main

`cmd/xfchat-bootstrapper/main.go` must stay thin.

Responsibilities:

- load default config
- open the state store
- build the orchestrator/runtime runner
- execute it
- exit with a clear fatal error on failure

`main` must not contain product logic. It only wires dependencies.

### Orchestrator

`internal/orchestrator` must become the owner of runtime flow control.

Responsibilities:

- load resumable state
- decide which phase to execute next
- call the internal platform setup runner
- call the internal OAuth runner
- call internal validation
- update state transitions

The orchestrator must no longer depend on `internal/larkcli.Adapter` for the normal path.

### Platform Setup Runner

This runner owns the open-platform interactions:

- open app entry page
- create app
- capture `App ID`
- capture `App Secret`
- configure redirect whitelist
- configure scopes
- publish version

This runner may reuse the existing `internal/browser` workflow and selectors, but those pieces must evolve from a step-name scaffold into executable logic.

### OAuth Runner

This runner owns the login flow:

- open the authorization page
- wait for user/browser completion if needed
- receive the localhost callback
- mark auth success in state

This work must not be delegated to `lark-cli auth login`.

### Local Runtime State

The runtime should continue using `internal/state` as the source of truth.

Requirements:

- save before/after meaningful phase transitions
- remain crash-recoverable
- continue supporting backup recovery and atomic replacement

### Diagnostics

The diagnostics layer must continue to support:

- redacted logs
- support bundles
- browser screenshots
- browser HTML snapshots
- user/platform/retryable classification

This remains a core runtime dependency.

## Existing Packages And Their New Roles

### Packages To Keep As Core Runtime

- `internal/preflight`
- `internal/browser`
- `internal/state`
- `internal/diagnostics`
- `internal/orchestrator`

### Packages To Downgrade From Core Runtime

- `internal/larkcli/adapter.go`
- `internal/larkcli/installer.go`

These may stay in the codebase for:

- compatibility experiments
- optional config export
- future integration testing

But they must not be part of the normal startup path after this refactor.

## Required Behavior Change

On a clean machine where only `xfchat-bootstrapper` is installed:

- the program must start
- the program must not immediately fail because `lark-cli.exe` is missing
- the program must move into its own preflight/platform-setup flow

This is the key acceptance criterion for the refactor.

## Testing Strategy

### Regression Test

Add a regression test that proves the startup/orchestration path does not require `lark-cli` to exist.

The test should fail against the old runtime model and pass once the runtime depends only on internal components.

### Internal Flow Tests

Add or extend tests so they cover:

- preflight still runs
- saved state is threaded into the next internal phase
- phase transitions advance without external command execution
- validation is internal, not shell-based

### Browser Runner Tests

For the first correction, browser logic can still be partially mocked, but the orchestrator must call an internal browser-facing interface rather than `exec`.

### Windows Path Regression

Add at least one test or constructor assertion proving that Windows startup no longer relies on:

```text
AppData\Roaming\XfchatLarkCli\bin\lark-cli.exe
```

## Error Handling

If browser or auth work is not implemented enough to complete successfully yet, the runtime should still fail in a way that reflects the real missing behavior, not a missing external binary.

Good failure examples:

- platform automation step not implemented
- callback listener startup failed
- browser session could not be established

Bad failure example:

- could not find `lark-cli.exe`

## Rollout Plan

### Step 1

Remove `lark-cli` from the startup/orchestration hot path.

### Step 2

Replace external command execution with internal runner interfaces.

### Step 3

Implement minimal real internal execution for platform setup and validation, even if some steps are still stubbed behind explicit internal interfaces.

### Step 4

Verify the installed binary on a clean machine now fails only on real internal behavior gaps, not on missing external dependencies.

## Explicit Product Decisions

- The standalone bootstrapper is the primary product
- The runtime must not require external `lark-cli`
- The installer UX remains one-line install + immediate run
- Existing installer/release work stays in place
- Internal runners replace external CLI orchestration

## Implementation Boundary

This spec changes runtime architecture only. It does not redefine the distribution system, GitHub Releases flow, or package-manager integration. Those remain as designed in the installer distribution spec.
