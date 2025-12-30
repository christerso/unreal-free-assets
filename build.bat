@echo off
echo Building Unreal Free Assets Monitor...

:: Get dependencies
go mod tidy

:: Build for Windows (hide console window with -ldflags)
go build -ldflags="-H windowsgui" -o unreal-free-assets.exe .

if %ERRORLEVEL% EQU 0 (
    echo Build successful! Run unreal-free-assets.exe to start.
) else (
    echo Build failed!
    exit /b 1
)
