# Send an initialize then a tools/list call to w2e-mcp.exe and print the
# raw JSON-RPC responses we get back. The script writes one framed request
# per line, feeds them via stdin, and reads the server's stdout lines.
$ErrorActionPreference = 'Continue'

$exe = ".\bin\w2e-mcp.exe"
$linesToPrint = 40

$init = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"probe","version":"0.1"}}}'
$initialized = '{"jsonrpc":"2.0","method":"notifications/initialized"}'
$toolsList = '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
$part = "`r`n"
$message = $init + $part + $initialized + $part + $toolsList + $part

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $exe
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.UseShellExecute = $false
$psi.CreateNoWindow = $true
$psi.WorkingDirectory = (Get-Location).Path

$proc = New-Object System.Diagnostics.Process
$proc.StartInfo = $psi
[void]$proc.Start()

# Send the requests, then close stdin so the server exits cleanly.
$proc.StandardInput.Write($message)
$proc.StandardInput.Flush()
$proc.StandardInput.Close()

# Read whatever stdout lines arrive (best-effort).
$stdout = New-Object System.Text.StringBuilder
$stderrLines = New-Object System.Collections.ArrayList
$count = 0
while (-not $proc.StandardOutput.EndOfStream -and $count -lt $linesToPrint) {
  $line = $proc.StandardOutput.ReadLine()
  if ($line) {
    [void]$stdout.AppendLine($line)
    $count++
  }
  if ($proc.HasExited) { break }
}
while (-not $proc.StandardError.EndOfStream) {
  $line = $proc.StandardError.ReadLine()
  if ($line) { [void]$stderrLines.Add($line) }
  if ($proc.HasExited) { Start-Sleep -Milliseconds 100; if ($proc.StandardError.EndOfStream) { break } }
}

"--- stdout ---"
$stdout.ToString()
"--- stderr ---"
[string]::Join("`n", $stderrLines.ToArray())
