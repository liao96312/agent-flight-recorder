param([switch]$Full)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $projectRoot
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'go test ./... failed' }
    go test ./internal -run 'TestRunLeavesNoPlaintextSecretInSession|TestHTMLReportEscapesInjectionAndBoundsContent|TestVerifyLocatesFirstEventAnomaly|TestVerifyLocatesArtifactAnomaly|TestPlanClean|TestExecuteClean|TestVerifySessionReadsV1|TestVerifyRejectsFutureVersionsWithoutModifyingSession' -count=1
    if ($LASTEXITCODE -ne 0) { throw 'focused security tests failed' }

    if ($Full) {
        $env:AFR_RELEASE = '1'
        go test ./internal -run '^TestRelease' -count=1 -timeout 10m
        if ($LASTEXITCODE -ne 0) { throw 'release stress tests failed' }
        Remove-Item Env:AFR_RELEASE
        & $PSScriptRoot\benchmark.ps1
    }

    & $PSScriptRoot\build.ps1 -Version 0.1.0
    $expected = (Get-Content .\dist\SHA256SUMS).Split(' ')[0]
    $actual = (Get-FileHash -Algorithm SHA256 .\dist\afr-windows-amd64.exe).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "checksum mismatch: $actual != $expected"
    }

    if (Get-Command claude -ErrorAction SilentlyContinue) {
        claude plugin validate --strict $projectRoot
        if ($LASTEXITCODE -ne 0) { throw 'Claude marketplace validation failed' }
        claude plugin validate --strict (Join-Path $projectRoot 'plugins\afr')
        if ($LASTEXITCODE -ne 0) { throw 'Claude plugin validation failed' }
    } else {
        Write-Warning 'Claude Code is unavailable; run both plugin validators before release.'
    }

    if (Get-Command codex -ErrorAction SilentlyContinue) {
        $pluginsJSON = codex plugin list --marketplace personal --available --json
        if ($LASTEXITCODE -ne 0) { throw 'Codex marketplace listing failed' }
        $plugins = $pluginsJSON | ConvertFrom-Json
        $found = @($plugins.installed) + @($plugins.available) | Where-Object { $_.name -eq 'afr' }
        if (-not $found) {
            throw 'Codex marketplace personal does not expose afr'
        }
    } else {
        Write-Warning 'Codex is unavailable; validate the local marketplace before release.'
    }

    git diff --check
    if ($LASTEXITCODE -ne 0) { throw 'git diff --check failed' }
    Write-Host 'AFR release checks passed.'
} finally {
    Remove-Item Env:AFR_RELEASE -ErrorAction SilentlyContinue
    Pop-Location
}
