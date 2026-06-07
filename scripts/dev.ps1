# CopyFlow 本地开发启动脚本（Windows PowerShell）
# 用法: .\scripts\dev.ps1
# 前提: 本地 MySQL 已就绪，且 backend/config/config.yaml 中 dsn 已配置正确

Set-Location "$PSScriptRoot\..\backend"

Write-Host "==> 启动 API (新窗口) ..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; go run ./cmd/api"

Write-Host "==> 启动 Worker (新窗口) ..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; go run ./cmd/worker"

Write-Host "==> 启动前端 (新窗口) ..."
Set-Location "$PSScriptRoot\..\frontend"
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; npm run dev"

Write-Host "完成! API: http://localhost:8080  前端: http://localhost:5173"
