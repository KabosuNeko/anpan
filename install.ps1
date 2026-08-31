# Anpan Windows PowerShell Installer
$ErrorActionPreference = 'Stop'

$Repo = "KabosuNeko/anpan"
$BinName = "anpan.exe"
$InstallDir = "$env:LOCALAPPDATA\Programs\anpan"

# Detect architecture
$Arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}

$Asset = "anpan-windows-$Arch.zip"

Write-Host "→ Fetching latest version of anpan..." -ForegroundColor Cyan
try {
    $ReleaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
    $Version = $ReleaseInfo.tag_name.TrimStart('v')
} catch {
    $Version = $null
}

if ($Version) {
    $DownloadUrl = "https://github.com/$Repo/releases/download/v$Version/$Asset"
} else {
    $DownloadUrl = "https://github.com/$Repo/releases/latest/download/$Asset"
}

$TempZip = [System.IO.Path]::GetTempFileName() + ".zip"
$TempExtract = Join-Path ([System.IO.Path]::GetTempPath()) "anpan-install-$([System.Guid]::NewGuid())"

Write-Host "→ Downloading $Asset from $DownloadUrl ..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing

Write-Host "→ Extracting..." -ForegroundColor Cyan
Expand-Archive -Path $TempZip -DestinationPath $TempExtract -Force

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$SourceExe = Join-Path $TempExtract "anpan.exe"
$DestExe = Join-Path $InstallDir $BinName

Copy-Item -Path $SourceExe -Destination $DestExe -Force

# Cleanup temp files
Remove-Item -Path $TempZip -Force -ErrorAction SilentlyContinue
Remove-Item -Path $TempExtract -Recurse -Force -ErrorAction SilentlyContinue

# Ensure in PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "→ Adding $InstallDir to user PATH..." -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

Write-Host "`n✓ Successfully installed anpan to $DestExe" -ForegroundColor Green
Write-Host "Restart your terminal or run: anpan --help`n" -ForegroundColor White
& "$DestExe" --version
