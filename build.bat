@echo off
REM Windows用ビルドスクリプト
REM 使用方法: build.bat

echo Building claude-statusline for Windows...

set GOOS=windows
set GOARCH=amd64

go build -ldflags="-s -w" -o claude-statusline.exe .

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Build successful!
    echo Output: claude-statusline.exe
    echo.
    echo To install, copy to: %%USERPROFILE%%\.claude\claude-statusline.exe
    echo.
    echo Then add to %%USERPROFILE%%\.claude\settings.json:
    echo {
    echo   "statusLine": {
    echo     "type": "command",
    echo     "command": "%%USERPROFILE%%\\.claude\\claude-statusline.exe"
    echo   }
    echo }
) else (
    echo Build failed!
)
