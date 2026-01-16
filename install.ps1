# Claude Statusline インストールスクリプト (PowerShell)
# 使用方法: .\install.ps1

$ErrorActionPreference = "Stop"

Write-Host "=== Claude Statusline Installer ===" -ForegroundColor Cyan
Write-Host ""

# ビルド
Write-Host "Building..." -ForegroundColor Yellow
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o claude-statusline.exe .

if (-not $?) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

Write-Host "Build successful!" -ForegroundColor Green
Write-Host ""

# インストール先
$claudeDir = Join-Path $env:USERPROFILE ".claude"
$targetPath = Join-Path $claudeDir "claude-statusline.exe"
$settingsPath = Join-Path $claudeDir "settings.json"

# .claudeディレクトリ確認
if (-not (Test-Path $claudeDir)) {
    Write-Host "Creating $claudeDir..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $claudeDir | Out-Null
}

# コピー
Write-Host "Installing to $targetPath..." -ForegroundColor Yellow
Copy-Item "claude-statusline.exe" $targetPath -Force
Write-Host "Installed!" -ForegroundColor Green
Write-Host ""

# settings.json更新
$statusLineConfig = @{
    type = "command"
    command = $targetPath
}

if (Test-Path $settingsPath) {
    Write-Host "Updating settings.json..." -ForegroundColor Yellow
    $settings = Get-Content $settingsPath -Raw | ConvertFrom-Json
    
    # PSCustomObjectの場合の処理
    if ($null -eq $settings) {
        $settings = @{}
    }
    
    # statusLineを追加/更新
    $settings | Add-Member -NotePropertyName "statusLine" -NotePropertyValue $statusLineConfig -Force
    
    $settings | ConvertTo-Json -Depth 10 | Set-Content $settingsPath -Encoding UTF8
    Write-Host "Settings updated!" -ForegroundColor Green
} else {
    Write-Host "Creating settings.json..." -ForegroundColor Yellow
    @{
        statusLine = $statusLineConfig
    } | ConvertTo-Json -Depth 10 | Set-Content $settingsPath -Encoding UTF8
    Write-Host "Settings created!" -ForegroundColor Green
}

Write-Host ""
Write-Host "=== Installation Complete ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Restart Claude Code to see the new statusline!"
Write-Host ""

# テスト実行
Write-Host "Testing..." -ForegroundColor Yellow
& $targetPath
