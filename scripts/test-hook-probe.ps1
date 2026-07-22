$ErrorActionPreference = 'Stop'

$testRoot = Join-Path $env:TEMP ('afr-hook-probe-test-' + [Guid]::NewGuid().ToString('n'))
$oldPluginData = $env:PLUGIN_DATA

try {
    $env:PLUGIN_DATA = $testRoot
    $payload = @{
        session_id = 'private-session-value'
        turn_id = 'private-turn-value'
        hook_event_name = 'UserPromptSubmit'
        prompt = 'private-prompt-value'
        transcript_path = 'private-transcript-path'
    } | ConvertTo-Json -Compress
    $payload | powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File (Join-Path $PSScriptRoot '..\plugins\afr\hooks\probe.ps1')

    $output = Get-Content -Raw (Get-ChildItem -File (Join-Path $testRoot 'hook-probe') | Select-Object -First 1).FullName
    if ($output -match 'private-(session|turn|prompt|transcript)') { throw 'probe persisted sensitive input' }
    $record = $output | ConvertFrom-Json
    if ($record.hook_event_name -ne 'UserPromptSubmit' -or $record.prompt_length -ne 20 -or -not $record.session_fingerprint) { throw 'probe metadata mismatch' }
    Write-Output 'hook probe self-test: PASS'
}
finally {
    $env:PLUGIN_DATA = $oldPluginData
    if (Test-Path -LiteralPath $testRoot) { Remove-Item -LiteralPath $testRoot -Recurse -Force }
}
