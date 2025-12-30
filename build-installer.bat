@echo off
setlocal enabledelayedexpansion

echo ============================================
echo   Unreal Free Assets - Installer Builder
echo ============================================
echo.

:: Check for Go
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go is not installed or not in PATH
    pause
    exit /b 1
)

:: Check for Inno Setup
set "ISCC="
if exist "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" (
    set "ISCC=C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
) else if exist "C:\Program Files\Inno Setup 6\ISCC.exe" (
    set "ISCC=C:\Program Files\Inno Setup 6\ISCC.exe"
)

if "!ISCC!"=="" (
    echo [ERROR] Inno Setup 6 not found!
    echo.
    echo Please install Inno Setup from:
    echo   https://jrsoftware.org/isdl.php
    echo.
    echo After installing, run this script again.
    start "" "https://jrsoftware.org/isdl.php"
    pause
    exit /b 1
)

echo [1/3] Building Go executable...
go build -ldflags="-H windowsgui -s -w" -o unreal-free-assets.exe .
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go build failed!
    pause
    exit /b 1
)
echo       Done!
echo.

:: Create dist folder
if not exist dist mkdir dist

echo [2/3] Creating installer...
"!ISCC!" /Q installer.iss
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Installer creation failed!
    pause
    exit /b 1
)
echo       Done!
echo.

echo [3/3] Cleaning up...
echo       Done!
echo.

echo ============================================
echo   BUILD SUCCESSFUL!
echo ============================================
echo.
echo Installer created at:
echo   dist\UnrealFreeAssetsSetup-1.0.0.exe
echo.
echo You can distribute this file to install the app.
echo.

:: Open the dist folder
explorer dist

pause
