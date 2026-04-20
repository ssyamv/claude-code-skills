# Install Release Checklist

Use this checklist to validate the release URL helpers and the installer-facing documentation before publishing a new release.

## Release Checklist

- Run `./scripts/build-release.sh`.
- Confirm the three `dist/` artifacts exist:
  - `dist/xfchat-bootstrapper-darwin-arm64`
  - `dist/xfchat-bootstrapper-darwin-amd64`
  - `dist/xfchat-bootstrapper-windows-amd64.exe`
- upload them to a GitHub Release tag.
- Verify the installer commands work for `latest` and for a pinned tag.

## Installer Validation

- Confirm the README documents GitHub Releases assets for the installer.
- Confirm the README includes both `latest` and pinned version install commands.
- Confirm the smoke test doc includes installer validation steps for both latest and pinned flows.
- Confirm the install checklist references the same release asset names used by the build scripts.
