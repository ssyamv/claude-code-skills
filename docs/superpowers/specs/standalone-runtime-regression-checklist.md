# Standalone Runtime Regression Checklist

## Purpose

Verify `xfchat-bootstrapper` starts on a clean machine without depending on an external `lark-cli` binary.

## Checklist

1. Install `xfchat-bootstrapper` on a clean machine.
2. Launch it without any external `lark-cli` binary present (`lark-cli.exe` on Windows, `lark-cli` on macOS and Linux).
3. Confirm startup reaches internal runtime handling instead of failing while resolving an external binary path from `PATH` or an external install location.
4. Confirm any remaining failure is an internal runtime error, not a missing external binary.

## Pass / Fail Signal

- Pass: startup reaches the bootstrapper's internal runtime handling and does not stop because it could not resolve an external `lark-cli` binary path.
- Fail: startup errors while looking for `lark-cli.exe`, `lark-cli`, or another external install location.
