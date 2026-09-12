# 本地开发环境变量（Windows）
# 用法：. .\scripts\dev.ps1

$root = Split-Path $PSScriptRoot -Parent

$env:FONU_DATA_DIR = Join-Path $root ".data"
$env:FONU_SESSION_SECRET = "dev-secret-change-me"
$env:FONU_NGINX_BIN = "C:\Users\o\Downloads\nginx-1.30.4\nginx.exe"
$env:FONU_NGINX_MIME_TYPES = "C:\Users\o\Downloads\nginx-1.30.4\conf\mime.types"

Write-Host "FONU_DATA_DIR=$env:FONU_DATA_DIR"
Write-Host "FONU_NGINX_BIN=$env:FONU_NGINX_BIN"
