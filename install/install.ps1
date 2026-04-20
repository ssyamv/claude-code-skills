param(
  [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$OwnerRepo = "ssyamv/claude-code-skills"
$InstallDir = Join-Path $env:LocalAppData "Programs\XfchatBootstrapper"
$BinaryPath = Join-Path $InstallDir "xfchat-bootstrapper.exe"
$AssetName = "xfchat-bootstrapper-windows-amd64.exe"

function Normalize-PathEntry {
  param(
    [string]$PathValue
  )

  if ([string]::IsNullOrWhiteSpace($PathValue)) {
    return $null
  }

  $expanded = [Environment]::ExpandEnvironmentVariables($PathValue).Trim()
  if ([string]::IsNullOrWhiteSpace($expanded)) {
    return $null
  }

  return [System.IO.Path]::GetFullPath($expanded).TrimEnd("\")
}

function Test-PathEntry {
  param(
    [string]$PathValue,
    [string]$Candidate
  )

  $normalizedCandidate = Normalize-PathEntry -PathValue $Candidate
  if ($null -eq $normalizedCandidate) {
    return $false
  }

  foreach ($entry in $PathValue -split ";") {
    $normalizedEntry = Normalize-PathEntry -PathValue $entry
    if ($null -ne $normalizedEntry -and $normalizedEntry -ieq $normalizedCandidate) {
      return $true
    }
  }

  return $false
}

if ($env:PROCESSOR_ARCHITECTURE -ne "AMD64") {
  throw "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)"
}

if ($Version -eq "latest") {
  $release = Invoke-RestMethod -Headers @{ "User-Agent" = "xfchat-bootstrapper-installer" } -Uri "https://api.github.com/repos/$OwnerRepo/releases/latest"
  $Version = $release.tag_name
}

if ([string]::IsNullOrWhiteSpace($Version)) {
  throw "failed to resolve release version"
}

$Url = "https://github.com/$OwnerRepo/releases/download/$Version/$AssetName"
$TempPath = [System.IO.Path]::GetTempFileName()

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
try {
  Invoke-WebRequest -Uri $Url -OutFile $TempPath
  Move-Item -Force -Path $TempPath -Destination $BinaryPath
} finally {
  if (Test-Path -LiteralPath $TempPath) {
    Remove-Item -LiteralPath $TempPath -Force
  }
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not (Test-PathEntry -PathValue $userPath -Candidate $InstallDir)) {
  $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $InstallDir } else { "$userPath;$InstallDir" }
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
}

Write-Host "Installed xfchat-bootstrapper $Version to $BinaryPath"
& $BinaryPath
