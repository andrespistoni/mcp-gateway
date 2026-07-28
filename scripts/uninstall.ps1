param(
    [switch]$Purge,
    [string]$InstallDir = $env:MCP_GATEWAY_INSTALL_DIR
)

$ErrorActionPreference = "Stop"

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\mcp-gateway"
}

$Binary = Join-Path $InstallDir "mcp-gateway.exe"
if (-not (Test-Path -LiteralPath $Binary)) {
    $Command = Get-Command mcp-gateway.exe -ErrorAction SilentlyContinue
    if ($Command) {
        $Binary = $Command.Source
        $InstallDir = Split-Path -Parent $Binary
    }
}

if (Test-Path -LiteralPath $Binary) {
    & $Binary disable-daemon
    if ($LASTEXITCODE -ne 0) {
        throw "No se pudo retirar la tarea programada; el binario no fue eliminado."
    }
} else {
    schtasks.exe /Query /TN mcp-gateway *> $null
    if ($LASTEXITCODE -eq 0) {
        schtasks.exe /End /TN mcp-gateway *> $null
        schtasks.exe /Delete /TN mcp-gateway /F | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "No se pudo retirar la tarea programada."
        }
    }
}

if (Test-Path -LiteralPath $Binary) {
    Remove-Item -LiteralPath $Binary -Force
}

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath) {
    $NormalizedInstallDir = $InstallDir.Trim().Trim('"').TrimEnd("\")
    $PathEntries = @($UserPath -split ";" | Where-Object {
        $_ -and ($_.Trim().Trim('"').TrimEnd("\") -ine $NormalizedInstallDir)
    })
    [Environment]::SetEnvironmentVariable("Path", ($PathEntries -join ";"), "User")
}

if ((Test-Path -LiteralPath $InstallDir) -and -not (Get-ChildItem -LiteralPath $InstallDir -Force)) {
    Remove-Item -LiteralPath $InstallDir
}

if ($Purge) {
    $ConfigDir = Join-Path $HOME ".mcp-gateway"
    Remove-Item -LiteralPath (Join-Path $ConfigDir "mcp-downstreams.yaml") -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath (Join-Path $ConfigDir "mcp-downstreams.yaml.lock") -Force -ErrorAction SilentlyContinue
    if ((Test-Path -LiteralPath $ConfigDir) -and -not (Get-ChildItem -LiteralPath $ConfigDir -Force)) {
        Remove-Item -LiteralPath $ConfigDir
    }
    Write-Host "Configuración propia eliminada."
} else {
    Write-Host "Configuración conservada en $HOME\.mcp-gateway."
}

Write-Host "mcp-gateway desinstalado."
Write-Host "Los registros .mcp.json de proyectos y Claude se conservan; consulte el README para retirarlos."
