# PowerShell Installer Manual Checklist

1. Run `irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1 | iex`.
2. Run `&([scriptblock]::Create((irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1))) -Version v0.1.0` and confirm the pinned version path is honored.
3. Verify `%LocalAppData%\Programs\XfchatBootstrapper\xfchat-bootstrapper.exe` exists after installation.
4. Run the installer a second time with the same version and verify `%LocalAppData%\Programs\XfchatBootstrapper` is still present in the user PATH exactly once.
5. Verify the installer launches `%LocalAppData%\Programs\XfchatBootstrapper\xfchat-bootstrapper.exe` immediately after download.
