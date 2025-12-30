@echo off
echo Removing Unreal Free Assets Monitor from Windows Startup...

set "STARTUP_FOLDER=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"

del "%STARTUP_FOLDER%\Unreal Free Assets.lnk" 2>nul

if %ERRORLEVEL% EQU 0 (
    echo Successfully removed from startup!
) else (
    echo Shortcut not found or already removed.
)
pause
