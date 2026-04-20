# Xfchat Bootstrapper One-Line Installer Design

**Date:** 2026-04-20
**Status:** Draft approved for spec write
**Owner:** Codex

## Summary

Add a distribution layer on top of the existing `xfchat-bootstrapper` binary so ordinary employees can install and run it with a single command, without cloning the repository and without installing `Go`.

The distribution model is:

- host prebuilt binaries on GitHub Releases
- provide a Bash installer for macOS
- provide a PowerShell installer for Windows
- support both `latest` and explicit version installation
- install the binary into a user-local path
- update the user's PATH automatically
- immediately launch the installed bootstrapper after installation

This design does not replace the bootstrapper itself. It adds a release and installation path that makes the bootstrapper consumable by employees.

## Goals

- Provide one-line install-and-run commands for `macOS` and `Windows`
- Require no repository checkout
- Require no local `Go` installation
- Default to installing the latest release
- Also support installing a pinned release version
- Install into user-local paths that do not require admin privileges
- Update PATH automatically when allowed
- Run the installed bootstrapper immediately after installation

## Non-Goals

- Supporting Linux in the first release
- Supporting Homebrew, `winget`, or `MSI/pkg` installers in the first release
- Building a package manager ecosystem integration
- Adding server-side distribution infrastructure outside GitHub Releases
- Changing the bootstrapper's business logic in this spec

## User Experience

### macOS Latest

```bash
curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash
```

### macOS Pinned Version

```bash
curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash -s -- --version v0.1.0
```

### Windows Latest

```powershell
irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1 | iex
```

### Windows Pinned Version

```powershell
&([scriptblock]::Create((irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1))) -Version v0.1.0
```

### Default Behavior

When a user runs the one-line installer:

1. The installer detects platform and architecture
2. The installer resolves the target release version
3. The installer downloads the appropriate binary from GitHub Releases
4. The installer installs it into a user-local directory
5. The installer updates PATH in user configuration
6. The installer launches the installed bootstrapper immediately

If PATH changes are not active in the current shell session, the installer still executes the binary by absolute path so the first run succeeds immediately.

## Recommended Architecture

Implement four release/distribution pieces:

1. release build outputs
2. release publishing convention
3. Bash installer
4. PowerShell installer

The bootstrapper binary remains the core product. The installers are thin clients that resolve version, fetch the correct asset, install it, and launch it.

## Distribution Design

### Release Assets

Each GitHub Release should publish these platform-specific assets:

- `xfchat-bootstrapper-darwin-arm64`
- `xfchat-bootstrapper-darwin-amd64`
- `xfchat-bootstrapper-windows-amd64.exe`

The first release does not need archives if the installer is prepared to download raw binaries directly.

### Release Source

Use GitHub Releases in `ssyamv/claude-code-skills`.

### Version Resolution

The installer must support:

- `latest`
- an explicit tag such as `v0.1.0`

For `latest`, the installer resolves the most recent published release.
For an explicit version, the installer resolves the matching tag exactly.

## Install Paths

### macOS

Install the binary to:

```text
~/.local/bin/xfchat-bootstrapper
```

Reasons:

- user-writable
- common convention
- aligns with shell-based one-line installers

### Windows

Install the binary to:

```text
%LocalAppData%\Programs\XfchatBootstrapper\xfchat-bootstrapper.exe
```

Reasons:

- user-writable
- does not require elevation
- common convention for per-user applications

## PATH Update Behavior

### macOS

The installer may update:

- `~/.zshrc`
- `~/.bashrc`

Behavior:

- detect the current shell where possible
- update the most relevant rc file
- avoid duplicate PATH entries
- add only the exact line needed for `~/.local/bin`

The installer should append a clearly marked managed block or an idempotent export line.

### Windows

The installer should:

- update the user-level PATH
- avoid duplicate entries
- not require admin rights

The installer must not attempt system-wide PATH modification in the first release.

## Installer Responsibilities

### Shared Responsibilities

Both installers must:

- support default `latest`
- support explicit version input
- detect platform/architecture
- build the correct download URL
- download the release asset
- install to the correct location
- make the binary executable when applicable
- update PATH
- run the installed bootstrapper by absolute path
- print a user-readable success/failure summary

### Bash Installer

The Bash installer should:

- accept `--version <tag>` optionally
- fail fast on unsupported platform/arch
- use `curl` for metadata and binary download
- use GitHub Releases APIs or stable release URLs to resolve `latest`
- install to `~/.local/bin`
- update shell rc files idempotently

### PowerShell Installer

The PowerShell installer should:

- accept `-Version <tag>` optionally
- fail fast on unsupported platform/arch
- use `Invoke-RestMethod` / `Invoke-WebRequest` appropriately
- resolve `latest` through GitHub APIs or stable release URLs
- install to `%LocalAppData%\Programs\XfchatBootstrapper`
- update user PATH idempotently

## Release URL Strategy

The simplest release URL model is:

- explicit version:
  - `https://github.com/ssyamv/claude-code-skills/releases/download/<tag>/<asset>`
- latest:
  - query GitHub Releases API for latest tag, then compose the explicit asset URL

This is preferred over hardcoding `releases/latest/download/...` if asset availability and naming need to be validated before download.

## Error Handling

### Download/Resolution Failures

Examples:

- release tag not found
- asset missing for detected platform
- GitHub API unavailable
- network timeout

Behavior:

- stop with a concise error
- show the attempted version and asset name

### Install Failures

Examples:

- target directory not writable
- file move/copy failure
- PATH update failure

Behavior:

- if binary install succeeded but PATH update failed, still run the binary by absolute path
- print a one-time manual PATH fix instruction

### Runtime Failures

Examples:

- bootstrapper starts but fails during its own execution

Behavior:

- report that install succeeded but first run failed
- print the absolute path of the installed binary for reruns
- do not roll back installation automatically

## Security Requirements

- Only download from the project GitHub repository
- Do not use `sudo` / admin elevation in the first release
- Do not write outside user-writable install paths
- Keep PATH modification idempotent and narrowly scoped
- Do not download and execute arbitrary unvalidated filenames

If checksum assets are added later, the installers should be extended to verify them before installation.

## Build And Release Requirements

The repository must be able to produce the three release binaries with stable names.

The release build scripts should:

- build `darwin-arm64`
- build `darwin-amd64`
- build `windows-amd64`
- place outputs in a predictable `dist/` layout

The first release does not require fully automated GitHub Release publication, but the repository should make the asset naming stable so publication can be scripted later.

## Documentation Requirements

The README should include:

- the one-line install command for macOS latest
- the one-line install command for macOS pinned version
- the one-line install command for Windows latest
- the one-line install command for Windows pinned version
- a short explanation of install location and PATH behavior
- a note that `GitHub Releases` must contain the corresponding binaries

## Testing Strategy

### Unit Tests

Cover:

- release asset name resolution
- version string handling
- install-path construction
- PATH update idempotency logic

### Script Validation

Cover:

- shell syntax validation for Bash installer
- PowerShell syntax review and execution where environment permits
- dry-run or mocked download verification where possible

### Manual Validation

Required before adoption:

- run the macOS one-line latest installer on a clean machine or user profile
- run the macOS one-line pinned installer on a clean machine or user profile
- run the Windows one-line latest installer on a clean machine or user profile
- run the Windows one-line pinned installer on a clean machine or user profile

For each path verify:

- binary lands in the correct location
- PATH is updated correctly
- installed binary launches immediately

## Rollout Plan

### Release 1

- stable cross-platform release binary names
- Bash installer
- PowerShell installer
- README install commands
- manual GitHub Releases upload supported by current build outputs

### Later Improvements

- release checksum verification
- GitHub Actions release automation
- package-manager support
- installer uninstall/repair modes

## Explicit Product Decisions

- Distribution source is GitHub Releases
- Both `latest` and explicit version install are supported
- Default install behavior is install-to-user-path, update PATH, then run immediately
- Shell/PATH modification is allowed
- First release targets only macOS and Windows
- First release does not require package managers

## Implementation Boundary

This spec covers distribution and installation only. It does not redefine the bootstrapper's internal automation behavior. Task-level implementation details, exact file layout, and test code belong in the follow-up implementation plan.
