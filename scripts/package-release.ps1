param(
    [string]$BinaryPath = (Join-Path $PSScriptRoot '../build/bin/NetSentry.exe'),
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '../build/bin/release')
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $BinaryPath -PathType Leaf)) {
    throw "Executable not found: $BinaryPath"
}
if ((Get-Item -LiteralPath $BinaryPath).Length -eq 0) {
    throw 'Cannot publish an empty executable.'
}

New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
Copy-Item -LiteralPath $BinaryPath -Destination (Join-Path $OutputDirectory 'NetSentry-windows-amd64.exe')
Copy-Item -LiteralPath (Join-Path $PSScriptRoot '../LICENSE') -Destination (Join-Path $OutputDirectory 'LICENSE')

$checksums = foreach ($name in @('NetSentry-windows-amd64.exe', 'LICENSE')) {
    $hash = (Get-FileHash -LiteralPath (Join-Path $OutputDirectory $name) -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $name"
}
$checksums | Set-Content -LiteralPath (Join-Path $OutputDirectory 'SHA256SUMS.txt') -Encoding ascii
