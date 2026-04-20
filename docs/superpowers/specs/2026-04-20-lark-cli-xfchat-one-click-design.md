# Lark CLI Xfchat One-Click Bootstrapper Design

**Date:** 2026-04-20
**Status:** Draft approved for spec write
**Owner:** Codex

## Summary

Build a one-click bootstrapper for `macOS` and `Windows` that automates the full setup flow for 讯飞私有化飞书 `lark-cli` usage against `xfchat.iflytek.com`. The bootstrapper must assume the employee is already logged into the company account in the default browser and then complete these steps with no manual terminal interaction:

1. Open the 讯飞开放平台 and create an app
2. Extract `App ID` and `App Secret`
3. Configure Redirect URL whitelist
4. Configure required permission scopes
5. Publish a new app version
6. Install a usable `lark-cli` binary locally
7. Initialize local `lark-cli` config with the created app credentials
8. Launch OAuth login and complete authorization in the browser
9. Validate the setup and report a clear result

The goal is to replace the current skill's multi-step manual workflow with a single executable that ordinary employees can run without understanding `Go`, `git`, command-line arguments, or the open platform configuration model.

## Goals

- Provide a single executable for `macOS` and `Windows`
- Require no manual shell commands from employees
- Avoid requiring local developer tooling such as `Go`, `Python`, or `git`
- Automate browser-side platform operations on `https://open.xfchat.iflytek.com`
- Automate local `lark-cli` install, config, OAuth, and validation
- Produce actionable diagnostics when the flow fails
- Support rerunning safely after partial completion

## Non-Goals

- Supporting Linux in the first release
- Supporting arbitrary Lark brands beyond `xfchat.iflytek.com`
- Supporting employee-specific custom scope sets in the first release
- Replacing the official `lark-cli` itself
- Managing enterprise accounts, RBAC, or employee directory state
- Building a server-side control plane in the first release

## Users And Assumptions

### Primary User

An ordinary company employee who needs working `lark-cli` access for 讯飞私有化飞书 and is not expected to understand the open platform or local CLI configuration details.

### Assumptions

- The user is already logged into company SSO in the default browser
- The user has permission to create an app in `https://open.xfchat.iflytek.com/app`
- The app creation flow, base info page, safe page, auth page, and publish flow are reachable through stable browser interactions
- Redirect URL whitelist can be configured by browser automation
- Required permission scopes can be configured by browser automation
- App publish can be triggered by browser automation
- `lark-cli auth login` can complete through a localhost callback on port `8080`
- The company can provide either embedded binaries or a fixed trusted download endpoint for `lark-cli`

## Product Requirements

### Entry Point

The employee runs one executable. The executable may be launched by double-click or command line. It must not require additional flags for the normal path.

### Zero Manual Terminal Interaction

For the standard success path, the user must not need to:

- clone repositories
- compile anything
- copy and paste `App ID`
- copy and paste `App Secret`
- manually type `lark-cli config init`
- manually open the safe/auth pages
- manually edit local config files

### Allowed Manual Actions

The user may still need to:

- approve or complete interactive login in the browser if the platform or OAuth flow demands it
- grant browser permissions if the OS prompts for them
- rerun the executable after an external issue is resolved

These are acceptable because they are browser approvals or OS confirmations rather than configuration work.

## Recommended Architecture

Use a local single-process bootstrapper with five modules:

1. `orchestrator`
2. `browser automation`
3. `lark-cli installer`
4. `lark-cli adapter`
5. `diagnostics`

This architecture keeps all first-release logic on the client machine and avoids introducing a server-side dependency. It also preserves a direct mapping to the existing manual skill flow, which reduces domain ambiguity.

## Alternative Approaches Considered

### Approach A: Browser Automation + Local Bootstrapper

Automate both the open platform and local CLI setup from one local executable.

**Pros**
- Matches the approved user experience
- No server-side dependency
- Can be delivered incrementally

**Cons**
- Sensitive to open platform UI changes
- Requires robust retry and selector maintenance

### Approach B: Server-Side Control Plane + Thin Client

A central service creates apps and returns credentials to the local executable.

**Pros**
- More stable long term
- Easier to update behavior without redistributing the client

**Cons**
- Requires new backend infrastructure
- Not suitable for first delivery

### Approach C: Manual Platform Setup + Automated Local Setup

Leave app creation and permission setup manual, automate only installation and OAuth.

**Pros**
- Faster to build
- Lower browser automation complexity

**Cons**
- Fails the approved “one-click” goal

### Recommendation

Implement **Approach A** in the first release, but explicitly avoid compiling `lark-cli` on employee machines. The bootstrapper should use prebuilt binaries supplied by the company or embedded in the distribution.

## System Design

### Phase A: Environment Inspection

The bootstrapper performs startup checks before changing state.

Checks:

- Detect platform: `macOS` or `Windows`
- Detect writable install directory
- Detect whether port `8080` is free
- Detect whether the browser can be launched
- Detect whether an existing `lark-cli` installation is present
- Detect whether a previous bootstrap state file exists

Outcomes:

- Continue automatically if all checks pass
- Offer automatic cleanup of stale local bootstrap state
- Fail fast with a user-readable message if the environment is fundamentally incompatible

### Phase B: Open Platform Automation

The browser automation module drives these flows on `open.xfchat.iflytek.com`:

1. Open app creation entry page
2. Create a new app
3. Navigate to app base info page
4. Read `App ID`
5. Reveal and read `App Secret`
6. Navigate to app safe page
7. Ensure `http://localhost:8080/callback` exists in Redirect URL whitelist
8. Navigate to app auth page
9. Enable a fixed company-approved scope set
10. Publish a new version

Artifacts captured from this phase:

- `appId`
- `appSecret`
- `brand=xfchat.iflytek.com`
- `appManagementUrl`
- timestamps for each completed substep
- screenshots and page HTML snapshots on failure

### Phase C: Local lark-cli Provisioning

The installer module must not depend on `Go`, source checkout, or `make`.

Supported provisioning methods:

1. Embedded prebuilt `lark-cli` binaries inside the app package
2. Download from a fixed trusted internal URL with checksum verification

Not supported in the normal path:

- cloning `https://code.iflytek.com/...`
- compiling from source on employee machines

The bootstrapper installs the binary into a user-local location.

Suggested paths:

- `macOS`: `~/Library/Application Support/XfchatLarkCli/bin/lark-cli`
- `Windows`: `%LocalAppData%\\XfchatLarkCli\\bin\\lark-cli.exe`

Optional convenience shims may also be created:

- `macOS`: symlink in `~/.local/bin` when possible
- `Windows`: optional wrapper `.cmd` in a user bin directory if the environment allows

The executable does not need to mutate global machine-level `PATH` in the first release if it can always invoke the installed binary via its full path.

### Phase D: Local Config Initialization

The adapter module runs local `lark-cli` commands with a controlled environment.

Required behavior:

- force `LARK_CLI_NO_PROXY=1`
- clear incompatible previous config if needed
- run `config init` using:
  - `--brand xfchat.iflytek.com`
  - generated `--app-id`
  - `--app-secret-stdin`

Validation after config:

- `config show` reports `brand=xfchat.iflytek.com`
- the stored config references the newly created app

### Phase E: OAuth Login

The bootstrapper starts `lark-cli auth login`, opens the browser if needed, waits for the localhost callback, and verifies successful login.

Expected flow:

1. Start `lark-cli auth login`
2. Ensure browser reaches the authorization page
3. Wait for callback completion on `http://localhost:8080/callback`
4. Run `auth status`
5. Record authenticated identity

If the user is not actually logged in despite the default assumption, the tool may block until login completes in the browser, but it must still keep the flow within the single executable session.

### Phase F: Final Validation

The bootstrapper runs a final validation bundle:

- verify installed binary exists
- verify `config show`
- verify `auth status`

The first release acceptance gate is exactly `config show + auth status`. It does not execute any additional read-only API smoke check.

## State Management And Resume

The tool must persist bootstrap progress locally so that reruns can resume instead of restarting from scratch when safe.

Persisted state must include:

- current phase
- current app URL
- `appId`
- whether `appSecret` has been captured
- whether safe page update succeeded
- whether auth scopes update succeeded
- whether publish succeeded
- local binary install path
- last error classification
- diagnostic artifact paths

Resume rules:

- If an app has already been created and is still usable, reuse it instead of creating a new one
- If local installation is already valid, skip reinstall unless the binary is missing or corrupt
- If OAuth already succeeded and config matches current app credentials, short-circuit to final validation

## Scope Management

The first release must use a fixed company-approved scope set packaged with the bootstrapper.

Reasoning:

- The user explicitly wants the script to handle scope configuration
- Allowing per-user custom scopes would complicate the UI and flow
- The browser automation is easier to validate against a known scope list

The scope list should be represented as configuration data rather than hard-coded inside selector logic so future updates do not require browser step rewrites.

## Browser Automation Requirements

The browser automation layer must support:

- launching the default browser or an automation-managed browser context
- waiting for authenticated pages to load
- interacting with buttons, inputs, menus, modals, and publish flows
- extracting visible values and hidden form outputs
- screenshot capture
- HTML snapshot capture
- resilient waits and retryable selectors

Design principles:

- Separate selector definitions from flow logic
- Prefer semantic and visible-text selectors before brittle DOM indexes
- Annotate every browser step with a diagnostic name
- Capture a screenshot before aborting a phase

## Security Requirements

- Never log raw `App Secret` in normal logs
- If `App Secret` is persisted for resume, store it encrypted or avoid persistence entirely and instead rely on the already-written `lark-cli` config state
- Use stdin for `--app-secret-stdin`
- Avoid writing secrets into shell history
- Avoid machine-global environment mutations where not required
- Treat browser-captured secrets as sensitive diagnostic data and redact them from exported logs

## Error Handling

All failures must be classified into one of three categories:

### Retryable

Examples:

- page element not found after a transient load delay
- network timeout
- download failure
- temporary browser launch failure
- temporary port conflict that clears after retry

Behavior:

- automatic retries with bounded backoff
- preserve diagnostics between attempts

### User-Actionable

Examples:

- browser not logged in
- user account lacks permission to create apps
- browser approval modal requires user confirmation
- local folder permissions block installation

Behavior:

- show a concise message describing the required employee action
- keep enough state to resume after rerun

### Platform-Actionable

Examples:

- open platform page structure changed
- publish flow changed materially
- fixed scope list is no longer accepted
- downloaded `lark-cli` version is incompatible

Behavior:

- stop
- emit a support-ready diagnostic bundle
- clearly tell the employee to contact IT/platform support

## Diagnostics

The bootstrapper must produce a diagnostic bundle on failure containing:

- phase and substep name
- timestamp
- platform and OS version
- binary version of the bootstrapper
- installed `lark-cli` version if available
- current app management URL if available
- last browser URL
- redacted logs
- screenshots
- HTML snapshots when useful
- stdout/stderr from `lark-cli` commands with secrets removed

Diagnostics should be written to a user-local support directory and referenced in the final failure message.

## Distribution Requirements

The artifact must be a single executable per platform distribution stream.

Two acceptable packaging patterns:

1. one `macOS` single executable and one `Windows` single executable
2. one logical product release containing both platform-specific single executables

The user-facing requirement is still “single executable” because each employee receives exactly one file appropriate for their machine.

## Testing Strategy

### Unit Tests

Cover:

- OS/platform path resolution
- bootstrap state transitions
- error classification
- config generation
- secret redaction
- retry policy decisions

### Mock Integration Tests

Cover:

- fake browser flows for app creation and page navigation
- fake `lark-cli` process interactions
- resume after partial failure
- publish failure handling

### Real End-To-End Validation

Required before release:

- one successful run on `macOS`
- one successful run on `Windows`
- one forced-failure diagnostic validation case on each platform

## Rollout Plan

### Release 1

- single executable bootstrapper
- embedded or trusted-download `lark-cli`
- browser automation for create app, safe, auth, publish, OAuth
- local config and validation
- diagnostics and resume support

### Later Improvements

- central service for selector/config updates
- remote config for scope set changes
- stronger smoke tests against approved read-only APIs
- admin mode for repair or uninstall

## Open Questions Resolved In This Spec

- **Can browser interaction be part of the one-click flow?** Yes
- **Are `App ID` and `App Secret` created through the web flow rather than pre-provisioned?** Yes
- **Can Redirect URL setup be browser-automated?** Yes
- **Can scope configuration be browser-automated?** Yes
- **Should this support `macOS + Windows`?** Yes
- **Should the deliverable be a single executable?** Yes
- **Should the standard path assume the employee is already logged in?** Yes

## Explicit Product Decisions

- First release is client-only, no backend control plane
- First release targets only `xfchat.iflytek.com`
- First release uses a fixed scope set
- First release avoids source compilation on employee machines
- Final success criteria are local config validity plus successful OAuth status

## Implementation Boundary

This spec defines the product and technical design for the one-click bootstrapper only. It does not yet define task-level implementation steps, exact file paths, test case code, or commit sequencing. Those belong in the implementation plan that follows after spec approval.
