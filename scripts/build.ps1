param([string]$Version = '0.1.0')

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$outputDirectory = Join-Path $projectRoot 'dist'
$output = Join-Path $outputDirectory 'afr-windows-amd64.exe'
$checksumOutput = Join-Path $outputDirectory 'SHA256SUMS'
$commit = (git -C $projectRoot rev-parse --short=12 HEAD).Trim()
$buildTime = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-s -w -X main.version=$Version -X main.commit=$commit -X main.buildTime=$buildTime"

New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null
$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
Push-Location $projectRoot
try {
    go build -trimpath -ldflags $ldflags -o $output ./cmd/afr
} finally {
    Pop-Location
}

if (-not (Test-Path -LiteralPath $output -PathType Leaf)) {
    throw 'Windows executable was not created'
}

& $output version
$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $output).Hash.ToLowerInvariant()
Set-Content -LiteralPath $checksumOutput -Encoding ascii -NoNewline -Value "$hash  afr-windows-amd64.exe`n"
Get-Content -LiteralPath $checksumOutput
$output
