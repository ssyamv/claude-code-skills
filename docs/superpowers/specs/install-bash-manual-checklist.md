# Bash Installer Manual Checklist

1. Run `curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash`.
2. Run `curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash -s -- --version v0.1.0` and confirm the pinned version path is honored.
3. Verify `~/.local/bin/xfchat-bootstrapper` exists and is executable after installation.
4. Verify `~/.zshrc` and `~/.bashrc` each contain a single managed PATH block that adds `~/.local/bin`.
5. Verify the installer launches `~/.local/bin/xfchat-bootstrapper` by absolute path immediately after download.
