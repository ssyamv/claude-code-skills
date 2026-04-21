# Xfchat Bootstrapper Smoke Test

## Purpose

Verify the packaged `xfchat-bootstrapper` binary can be built and that the expected browser-based bootstrap flow still works after release packaging changes.

## Build Commands

Run one of the following from the repository root:

```bash
make build
```

```bash
./scripts/build-release.sh
```

```powershell
./scripts/build-release.ps1
```

## Smoke Test Steps

1. Launch the built binary on a machine with a logged-in Chrome or Edge profile.
2. Confirm an app is created under `https://open.xfchat.iflytek.com/app`.
3. Confirm `http://localhost:8080/callback` is present in the app's safe settings.
4. Confirm the required scopes are selected and the version is published.
5. Confirm `lark-cli config show` reports `xfchat.iflytek.com`.
6. Confirm `lark-cli auth status` succeeds.

## Installer Validation

1. macOS latest: confirm the one-line command fetches the installer script from `raw.githubusercontent.com`, then resolves and downloads the release binary from GitHub Releases.
2. macOS explicit tag: confirm the pinned one-line command fetches the installer script from `raw.githubusercontent.com`, then resolves and downloads the tagged release binary from GitHub Releases.
3. Windows latest: confirm the one-line command fetches the installer script from `raw.githubusercontent.com`, then resolves and downloads the release binary from GitHub Releases.
4. Windows explicit tag: confirm the pinned one-line command fetches the installer script from `raw.githubusercontent.com`, then resolves and downloads the tagged release binary from GitHub Releases.

## Expected Outputs

- `make test` completes successfully.
- Release binaries are written to `dist/`.
- The smoke test and build scripts do create `dist/` artifacts, so the worktree may be dirty until you run `make clean`.

## Standalone Runtime Regression Note

Before release, also verify the standalone runtime does not fail because an external `lark-cli` binary is missing (`lark-cli.exe` on Windows, `lark-cli` on macOS and Linux). Use [`docs/superpowers/specs/standalone-runtime-regression-checklist.md`](docs/superpowers/specs/standalone-runtime-regression-checklist.md) as the release regression checklist for that scenario.

## Real Browser Validation

Before release, also verify browser automation against the real open platform and a successful localhost callback flow.

## macOS API-First Verification

1. Start from a macOS machine with a logged-in Chrome or Edge profile for `open.xfchat.iflytek.com`.
2. Run the bootstrapper build that includes the API-first platform setup runner.
3. Confirm the setup phase succeeds without GUI page-clicking automation.
4. Confirm the diagnostics output includes no raw cookie, CSRF, or bearer token values.
5. Confirm OAuth opens the real authorization URL and the localhost callback succeeds.
