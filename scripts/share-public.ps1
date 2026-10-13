# Publica temporalmente el entorno local (http://localhost:8080) en una URL
# HTTPS de Cloudflare (trycloudflare.com) para que otras personas lo prueben.
# Requiere el stack levantado y cloudflared instalado. Ctrl+C lo cierra y
# devuelve el backend a http://localhost:8080.

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$compose = @('-f', "$root\compose.yaml", '-f', "$root\compose.mail.yaml", '--profile', 'app')

$cloudflared = (Get-Command cloudflared -ErrorAction SilentlyContinue).Source
if (-not $cloudflared) {
    $cloudflared = @("${env:ProgramFiles(x86)}\cloudflared\cloudflared.exe", "$env:ProgramFiles\cloudflared\cloudflared.exe") |
        Where-Object { Test-Path $_ } | Select-Object -First 1
}
if (-not $cloudflared) { throw 'No se encontró cloudflared. Instálalo con: winget install Cloudflare.cloudflared' }

try { Invoke-RestMethod http://localhost:8080/health/ready | Out-Null }
catch { throw 'La aplicación no responde en http://localhost:8080. Levanta primero los contenedores.' }

$log = Join-Path $env:TEMP 'tallerflow-tunnel.log'
Remove-Item $log -ErrorAction SilentlyContinue
$tunnel = Start-Process -FilePath $cloudflared -ArgumentList @('tunnel', '--no-autoupdate', '--url', 'http://localhost:8080') `
    -RedirectStandardError $log -WindowStyle Hidden -PassThru

function Set-BackendOrigin([string]$origin) {
    $env:TF_ALLOWED_ORIGIN = $origin
    $env:TF_PASSWORD_RESET_BASE_URL = $origin
    docker compose @compose up -d --wait backend | Out-Null
}

try {
    Write-Host 'Abriendo túnel...'
    $url = $null
    for ($i = 0; $i -lt 60 -and -not $url; $i++) {
        Start-Sleep -Seconds 1
        if ($tunnel.HasExited) { throw "cloudflared terminó inesperadamente. Revisa $log" }
        $match = Select-String -Path $log -Pattern 'https://[a-z0-9-]+\.trycloudflare\.com' -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($match) { $url = $match.Matches[0].Value }
    }
    if (-not $url) { throw "No se obtuvo la URL pública. Revisa $log" }

    Write-Host 'Configurando el backend para la URL pública...'
    Set-BackendOrigin $url

    Write-Host ''
    Write-Host "  TallerFlow público en: $url" -ForegroundColor Green
    Write-Host '  Comparte esa URL. Deja esta ventana abierta; Ctrl+C para cerrar.'
    Write-Host '  Mientras esté activo, http://localhost:8080 no acepta inicios de sesión: entra tú también por la URL pública.'
    Write-Host ''
    Wait-Process -Id $tunnel.Id
}
finally {
    if (-not $tunnel.HasExited) { Stop-Process -Id $tunnel.Id -Force }
    Write-Host 'Túnel cerrado. Restaurando el backend a http://localhost:8080...'
    Set-BackendOrigin 'http://localhost:8080'
    Remove-Item Env:TF_ALLOWED_ORIGIN, Env:TF_PASSWORD_RESET_BASE_URL -ErrorAction SilentlyContinue
}
