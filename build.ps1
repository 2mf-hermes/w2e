# build.ps1 — canonical build of the w2e Windows GUI + CLI binary.
#
# Output: bin\w2e.exe (PE32+, subsystem = GUI, no DOS console window).
# CLI subcommands (doctor/build/version/...) still print through the caller's
# console at runtime via cmd/w2e/console_windows.go (AttachConsole).
#
# This is the script to use for any Windows deployment. `go build .\cmd\w2e`
# without these flags would leave subsystem=CONSOLE and spawn a black DOS
# box behind the GUI — never do that.
#
# Usage:
#   .\build.ps1              # full rebuild, default version stamp
#   .\build.ps1 -Version 1.2.3
#   .\build.ps1 -Version 1.2.3 -AlsoMcp        # also build bin\w2e-mcp.exe
#
# Flags are also exposed for cross-platform: w2e's flags avoid assuming a CLI
#   shell, and the -H windowsgui flag is only emitted on Windows.
[CmdletBinding()]
param(
    [string]$Version = "1.0.0-dev",
    [string]$Commit   = "",
    [switch]$AlsoMcp
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSCommandPath
Set-Location $root
if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Path "bin" | Out-Null }

$ldflags = @("-X", "github.com/minfu/w2e/internal/version.Version=$Version")
if ($Commit) { $ldflags += "-X", "github.com/minfu/w2e/internal/version.Commit=$Commit" }
$ldflags += "-H", "windowsgui"   # GUI subsystem; no console window for the GUI
$ldArgs = ($ldflags -join " ")

Write-Host "[build] go build w2e.exe (version $Version, GUI subsystem)"
go build -ldflags "$ldArgs" -o "bin\w2e.exe" ".\cmd\w2e"
if ($LASTEXITCODE -ne 0) { throw "w2e.exe build failed (exit $LASTEXITCODE)" }

if ($AlsoMcp) {
    Write-Host "[build] go build w2e-mcp.exe (MCP stdio server)"
    # MCP server is non-GUI but still Windows: build it WITHOUT -H windowsgui
    # so it can run as a stdio child process in the AI agent's console.
    go build -o "bin\w2e-mcp.exe" ".\cmd\w2e-mcp"
    if ($LASTEXITCODE -ne 0) { throw "w2e-mcp.exe build failed (exit $LASTEXITCODE)" }
}

Write-Host "[build] OK -> bin\w2e.exe"
if ($AlsoMcp) { Write-Host "[build] OK -> bin\w2e-mcp.exe" }
