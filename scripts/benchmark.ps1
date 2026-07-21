$ErrorActionPreference = 'Stop'

$os = Get-CimInstance Win32_OperatingSystem
$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1
$disk = Get-CimInstance Win32_LogicalDisk -Filter "DeviceID='$((Get-Location).Drive.Name):'"

[pscustomobject]@{
    Windows = $os.Caption + ' ' + $os.Version
    CPU = $cpu.Name
    LogicalCPUs = $cpu.NumberOfLogicalProcessors
    MemoryGiB = [math]::Round($os.TotalVisibleMemorySize / 1MB, 2)
    Disk = $disk.DeviceID
    DiskFreeGiB = [math]::Round($disk.FreeSpace / 1GB, 2)
    Git = (git version)
    WorkspaceFixture = '10000 files x 1024 bytes; 100 modified files'
    EventFixture = '100000 events x 1024-byte payload'
    FlushMode = '64 KiB or 1 second; critical events sync immediately'
} | Format-List

$env:AFR_PERF = '1'
go test ./internal -run TestLargeRepositoryPerformance -count=1 -v
if ($LASTEXITCODE -ne 0) {
    throw 'performance fixture failed'
}
