@echo off
REM Build script for LAN File Share (Windows)
echo Building LAN File Share...
go build -ldflags="-s -w" -o lan-file-share.exe
if %ERRORLEVEL% EQU 0 (
    echo Build successful! Run: lan-file-share.exe --dir .\shared
) else (
    echo Build failed.
)
