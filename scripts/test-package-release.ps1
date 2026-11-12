$ErrorActionPreference = 'Stop'
$packageScript = Join-Path $PSScriptRoot 'package-release.ps1'
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $temp | Out-Null
try {
    $binary = Join-Path $temp 'input.exe'
    [System.IO.File]::WriteAllText($binary, 'abc', [System.Text.Encoding]::ASCII)
    $output = Join-Path $temp 'release'

    & $packageScript -BinaryPath $binary -OutputDirectory $output
    $asset = Join-Path $output 'NetSentry-windows-amd64.exe'
    if ([System.IO.File]::ReadAllText($asset) -cne 'abc') {
        throw 'Packaging changed the executable contents.'
    }
    $checksums = Get-Content (Join-Path $output 'SHA256SUMS.txt')
    if ($checksums -cnotcontains 'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad  NetSentry-windows-amd64.exe') {
        throw 'The executable checksum or asset filename is incorrect.'
    }
    $license = Join-Path $output 'LICENSE'
    $sourceLicense = Join-Path $PSScriptRoot '../LICENSE'
    if ((Get-FileHash $license).Hash -ne (Get-FileHash $sourceLicense).Hash) {
        throw 'The release must include an unchanged license.'
    }
    $files = @(Get-ChildItem -LiteralPath $output -File)
    if ($files.Count -ne 3) {
        throw 'Unexpected release assets (possibly local runtime data).'
    }

    $rejected = $false
    try {
        & $packageScript -BinaryPath (Join-Path $temp 'missing.exe') -OutputDirectory (Join-Path $temp 'invalid')
    } catch {
        $rejected = $true
    }
    if (-not $rejected) { throw 'Packaging must reject a missing executable.' }
    Write-Output 'Release packaging tests passed.'
} finally {
    # Only delete the uniquely created test directory under the system temp root.
    $resolved = [System.IO.Path]::GetFullPath($temp)
    $root = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
    if (-not $resolved.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw 'Unsafe test cleanup path.'
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
