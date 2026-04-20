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
