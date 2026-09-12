# 清空本地数据目录（需先停止 fonu）
$root = Split-Path $PSScriptRoot -Parent
$data = Join-Path $root ".data"

Get-Process fonu -ErrorAction SilentlyContinue | Stop-Process -Force
Start-Sleep -Seconds 1

if (Test-Path $data) {
    Remove-Item -Recurse -Force $data
}
Write-Host "已清空 $data"
Write-Host "首次启动请访问 /setup 创建管理员账号（不要运行 cmd/seed）"
