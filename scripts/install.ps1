$ErrorActionPreference = "Stop"

$Repository = if ($env:MCP_GATEWAY_REPOSITORY) { $env:MCP_GATEWAY_REPOSITORY } else { "andrespistoni/mcp-gateway" }
$Version = if ($env:MCP_GATEWAY_VERSION) { $env:MCP_GATEWAY_VERSION } else { "latest" }
$InstallDir = if ($env:MCP_GATEWAY_INSTALL_DIR) { $env:MCP_GATEWAY_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\mcp-gateway" }

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "mcp-gateway requiere Windows de 64 bits."
}

$Architecture = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

if ($Version -eq "latest") {
    $Latest = Invoke-WebRequest -UseBasicParsing "https://github.com/$Repository/releases/latest"
    $Version = $Latest.BaseResponse.ResponseUri.Segments[-1].TrimEnd("/")
}

if ($Version.StartsWith("v")) {
    $Release = $Version.Substring(1)
} else {
    $Release = $Version
    $Version = "v$Version"
}

$Archive = "mcp-gateway_${Release}_windows_${Architecture}.zip"
$Checksums = "mcp-gateway_${Release}_checksums.txt"
$BaseUrl = "https://github.com/$Repository/releases/download/$Version"
$TempDir = Join-Path ([IO.Path]::GetTempPath()) ("mcp-gateway-" + [Guid]::NewGuid())

try {
    New-Item -ItemType Directory -Path $TempDir | Out-Null
    $ArchivePath = Join-Path $TempDir $Archive
    $ChecksumsPath = Join-Path $TempDir $Checksums
    Invoke-WebRequest -UseBasicParsing "$BaseUrl/$Archive" -OutFile $ArchivePath
    Invoke-WebRequest -UseBasicParsing "$BaseUrl/$Checksums" -OutFile $ChecksumsPath

    $ChecksumLine = Get-Content $ChecksumsPath | Where-Object { $_ -match "\s+$([Regex]::Escape($Archive))$" } | Select-Object -First 1
    if (-not $ChecksumLine) { throw "El manifiesto no contiene $Archive." }
    $Expected = ($ChecksumLine -split "\s+")[0].ToLowerInvariant()
    $Actual = (Get-FileHash -Algorithm SHA256 $ArchivePath).Hash.ToLowerInvariant()
    if ($Expected -ne $Actual) { throw "Checksum SHA-256 inválido." }

    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item (Join-Path $TempDir "mcp-gateway.exe") (Join-Path $InstallDir "mcp-gateway.exe") -Force

    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $PathEntries = @($UserPath -split ";" | Where-Object { $_ })
    if ($PathEntries -notcontains $InstallDir) {
        $NewPath = (@($PathEntries) + $InstallDir) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Host "Se añadió $InstallDir al PATH del usuario."
    }

    Write-Host "mcp-gateway $Release instalado en $InstallDir"
    Write-Host "Abra una terminal nueva y ejecute: mcp-gateway setup"
} finally {
    if (Test-Path $TempDir) { Remove-Item -Recurse -Force $TempDir }
}
