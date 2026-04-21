# Xfchat Bootstrapper macOS API-First Design

**Date:** 2026-04-21

## Goal

Complete the `macOS` bootstrap path by making the open-platform setup flow run through the real `open.xfchat.iflytek.com` HTTP interfaces instead of browser GUI automation, while preserving the current cross-platform structure so existing `Windows` support does not regress.

This design changes the primary execution model from:

- browser-driven page clicking

to:

- browser-backed session bootstrap
- session-backed platform HTTP client
- callback-server-based OAuth completion

`macOS` is the only platform that must be proven end-to-end in this round. `Windows` remains in scope for code compatibility, builds, and existing tests, but is not a required real-environment validation target for this milestone.

## Non-Goals

This round does not include:

- building a Windows-specific real-environment completion path
- maintaining browser GUI automation as the primary success path
- adding automatic fallback from HTTP execution back to page-clicking automation
- supporting only official documented APIs if the site relies on internal authenticated endpoints

## Product Decision

The bootstrapper is allowed to reverse and reuse the actual authenticated HTTP requests used by the XFChat open-platform web app.

The browser remains necessary, but only to reuse the employee's already logged-in local Chrome or Edge profile and extract the authenticated session context required by the HTTP client.

The browser is no longer the main business executor for:

- app creation
- redirect URL configuration
- scope configuration
- version creation or publication

## User Outcome

On `macOS`, a logged-in employee should be able to run `xfchat-bootstrapper` and have it:

1. attach to the local logged-in Chrome or Edge profile
2. bootstrap authenticated session context from `open.xfchat.iflytek.com`
3. create and configure the app through HTTP requests
4. initiate and complete OAuth with the localhost callback server
5. persist bootstrap state showing a successful authenticated result

The user should not need to manually click through the developer console UI during the normal success path.

## Architecture

The runtime should be reorganized around four explicit roles.

### 1. Browser Session Bootstrap

Responsibility:

- resolve the local logged-in Chrome or Edge profile
- open the XFChat open-platform site in a real browser context
- wait until the authenticated site session is stable
- capture reusable authenticated request context

Expected captured context may include:

- cookies
- CSRF token
- anti-forgery token
- origin or referer requirements
- any request headers that are mandatory for the site's internal APIs

This module must not own business workflow steps like "click create app" or "click publish".

### 2. Platform HTTP Client

Responsibility:

- call the real site interfaces needed for open-platform setup
- accept the session context extracted by the browser bootstrap phase
- expose narrow methods for business operations

The client must provide explicit operations for:

- `CreateApp`
- `GetAppCredentials`
- `EnsureRedirectURL`
- `EnsureScopes`
- `CreateVersion`
- `PublishVersion`
- any required read-before-write or lookup operations needed to make those idempotent

The client should be organized around business intents rather than raw endpoint wrappers, but diagnostics must still preserve enough request and response metadata to debug contract drift.

### 3. Platform Setup Runner

Responsibility:

- orchestrate session bootstrap and HTTP client calls
- return normalized platform setup metadata
- persist enough state for resume and diagnostics

The platform setup runner becomes the main owner of:

- app creation
- app credential capture
- callback configuration
- scope enablement
- version publication

Its output must include:

- `AppID`
- `AppSecret` if the site exposes it in this flow
- `AppURL`
- `AuthURL` or enough metadata to construct the authorization URL deterministically

### 4. OAuth Runner

Responsibility:

- start the localhost callback server
- open the actual authorization URL in the logged-in browser profile
- wait for callback completion
- map callback success or failure into runtime errors and final state

This phase remains browser-backed because the authorization interaction is still fundamentally tied to the employee's authenticated browser session.

## Data Flow

### Phase 1: Preflight

Preflight remains bootstrapper-owned.

It must verify:

- current platform is `darwin` or `windows`
- default browser is `Chrome` or `Edge`
- reusable browser profile can be resolved
- callback port is available
- installation root is writable

No XFChat business logic belongs in preflight.

### Phase 2: Platform Setup

`macOS` success path:

1. resolve local browser profile
2. launch a browser session using that profile
3. visit the open-platform site
4. extract authenticated session context
5. execute open-platform setup through HTTP calls
6. normalize response data into bootstrap state

At the end of this phase, bootstrap state must contain enough information for OAuth to proceed without reopening the setup UI flow.

### Phase 3: OAuth

1. start localhost callback server
2. generate or retrieve the actual authorization URL
3. open authorization URL with the same logged-in browser profile
4. wait for callback result
5. mark `AuthSuccess=true` if callback completes successfully

### Phase 4: Validate

Validation remains bootstrapper-owned and must depend only on bootstrapper state plus any strictly necessary local config files.

Minimum success requirement for this round:

- `AppID != ""`
- `AuthSuccess == true`

## Interface Discovery Strategy

The platform HTTP client must not be built from guessed endpoints.

The first implementation pass should use a controlled observation mode on `macOS`:

1. open the target developer pages with a logged-in browser profile
2. capture actual network requests through CDP
3. identify the request contract for each required business operation
4. translate that contract into stable internal client methods

The captured contract must record at least:

- HTTP method
- path
- required query parameters
- required request body fields
- required cookies or headers
- success response fields
- relevant failure response shape

This information should be recorded in implementation notes or diagnostics-oriented docs rather than left only in transient chat context.

## Execution Policy

This milestone is `API-first` and not `GUI-with-fallback`.

That means:

- HTTP execution is the primary and only required business path
- browser GUI automation is not allowed to silently take over when HTTP calls fail
- HTTP failures must surface clearly so contract drift is visible

Browser-based interaction is still allowed for:

- attaching to logged-in session state
- opening the final authorization URL
- collecting observation data during interface discovery

## Error Model

Errors should be classified into three operational groups.

### Session Bootstrap Errors

Examples:

- browser profile not found
- browser launch failure
- user not logged into XFChat
- cookies or tokens missing
- open-platform home page never reaches authenticated state

Required handling:

- fail fast
- produce a clear user-facing reason
- include diagnostics that distinguish browser/profile failure from site-contract failure

### Platform API Contract Errors

Examples:

- `401` or `403` from internal site endpoints
- request rejected due to missing CSRF or anti-forgery data
- unexpected response schema
- required endpoint cannot be found or no longer behaves as observed

Required handling:

- save redacted request and response metadata
- stop the flow rather than falling back to GUI automation
- make the failure actionable for maintainers

### Business-State Errors

Examples:

- app created but credentials cannot be extracted
- callback URL update did not persist
- one or more required scopes remain disabled
- version creation succeeded but publish did not
- OAuth callback returns an explicit error

Required handling:

- persist the most recent recoverable state
- keep failure semantics aligned with resumable orchestration
- ensure diagnostics show which business step failed

## macOS-First Validation Requirement

This milestone is only complete after one real `macOS` environment run succeeds end-to-end.

The real run must prove:

- logged-in Chrome or Edge profile reuse
- successful session bootstrap
- successful app creation through HTTP
- successful callback URL configuration
- successful required-scope enablement
- successful version creation and publication
- successful OAuth callback completion
- successful final validation

The real run should also record:

- browser used
- date of verification
- observed endpoint families used by the implementation
- any site-specific quirks discovered during execution

## Windows Compatibility Requirement

Even though `Windows` is not the primary validation target in this milestone, this work must preserve:

- current build support
- current test coverage expectations
- current package boundaries and platform branching discipline

This round must not:

- hardcode `darwin` assumptions directly into business workflow types
- break Windows profile resolution
- remove Windows code paths from preflight, session setup, or packaging

Platform-specific branching should remain localized to:

- browser/profile resolution
- browser launch behavior
- filesystem conventions

The HTTP client and orchestration layers should stay platform-neutral wherever possible.

## Code Organization Guidance

The code should move away from treating `internal/browser` as the owner of the full platform setup workflow.

Preferred direction:

- keep browser-profile and browser-session concerns in `internal/browser`
- introduce a dedicated open-platform HTTP client package for XFChat platform operations
- move platform business orchestration into a runner that depends on both browser-session bootstrap and the HTTP client

The current selector-heavy path should stop growing as the primary implementation. Existing selector code may remain temporarily for observation or compatibility, but new business logic should not be added to it unless required for session bootstrap or controlled debugging.

## Testing Strategy

Testing should cover four layers.

### Unit Tests

- browser profile resolution
- session-context extraction normalization
- HTTP request construction
- response parsing
- idempotent scope and redirect reconciliation logic
- OAuth callback handling

### Contract-Oriented Tests

- validate that client methods fail clearly when required cookies or tokens are missing
- validate request builders include the headers and body fields discovered from the real site
- validate parsing against representative success and error payloads

### Orchestrator Tests

- platform setup returns normalized state
- state advances from `platform_setup` to `oauth` to `validate`
- resume semantics remain correct when failures occur after partial progress

### Real macOS Smoke Test

At least one manual but evidence-backed real run must be documented for this milestone.

## Acceptance Criteria

This design is complete when all of the following are true:

- `macOS` uses browser-backed session bootstrap plus HTTP-based platform execution as the main path
- the bootstrapper no longer depends on GUI page clicking to complete platform setup
- app creation and configuration complete through observed real site interfaces
- OAuth completes through the localhost callback server
- bootstrap state ends with `AppID` populated and `AuthSuccess=true`
- one real `macOS` run is documented successfully
- existing Windows code structure and tests do not regress

## Implementation Boundary For The Next Plan

The implementation plan for this design should focus on:

- extracting authenticated session context from the logged-in browser profile
- introducing the XFChat platform HTTP client
- switching the platform setup phase to API-first execution
- tightening diagnostics for request and response contract failures
- documenting and verifying one real `macOS` smoke run

The implementation plan should not expand scope into:

- full Windows real-environment completion
- GUI fallback automation
- unrelated installer redesign
