$ErrorActionPreference = 'Stop'

$raw = [Console]::In.ReadToEnd()
$inputObject = $raw | ConvertFrom-Json

function Get-Fingerprint([object]$Value) {
    if ($null -eq $Value) { return $null }
    $bytes = [Text.Encoding]::UTF8.GetBytes([string]$Value)
    $hash = [Security.Cryptography.SHA256]::Create()
    try { return ([BitConverter]::ToString($hash.ComputeHash($bytes))).Replace('-', '').ToLowerInvariant() }
    finally { $hash.Dispose() }
}

$fieldTypes = [ordered]@{}
foreach ($property in $inputObject.PSObject.Properties | Sort-Object Name) {
    $fieldTypes[$property.Name] = if ($null -eq $property.Value) { 'null' } else { $property.Value.GetType().Name }
}

$record = [ordered]@{
    observed_at = [DateTime]::UtcNow.ToString('o')
    hook_event_name = $inputObject.hook_event_name
    session_fingerprint = Get-Fingerprint $inputObject.session_id
    turn_fingerprint = Get-Fingerprint $inputObject.turn_id
    source = $inputObject.source
    tool_name = $inputObject.tool_name
    permission_mode = $inputObject.permission_mode
    model = $inputObject.model
    field_types = $fieldTypes
    prompt_length = if ($null -eq $inputObject.prompt) { $null } else { ([string]$inputObject.prompt).Length }
    tool_input_bytes = if ($null -eq $inputObject.tool_input) { $null } else { [Text.Encoding]::UTF8.GetByteCount(($inputObject.tool_input | ConvertTo-Json -Compress -Depth 20)) }
    tool_response_bytes = if ($null -eq $inputObject.tool_response) { $null } else { [Text.Encoding]::UTF8.GetByteCount(($inputObject.tool_response | ConvertTo-Json -Compress -Depth 20)) }
    last_assistant_message_length = if ($null -eq $inputObject.last_assistant_message) { $null } else { ([string]$inputObject.last_assistant_message).Length }
}

$dataRoot = if ($env:PLUGIN_DATA) { $env:PLUGIN_DATA } else { Join-Path $env:TEMP 'afr-plugin-data' }
$probeRoot = Join-Path $dataRoot 'hook-probe'
[IO.Directory]::CreateDirectory($probeRoot) | Out-Null
$path = Join-Path $probeRoot (([Guid]::NewGuid().ToString('n')) + '.json')
[IO.File]::WriteAllText($path, ($record | ConvertTo-Json -Depth 10), [Text.UTF8Encoding]::new($false))
