@echo off
REM Windows用ビルドスクリプト
REM 使用方法: build.bat

echo Building claude-statusline for Windows...

set GOOS=windows
set GOARCH=amd64

go build -ldflags="-s -w" -o claude-statusline.exe .
