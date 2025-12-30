@echo off
echo Adding Unreal Free Assets Monitor to Windows Startup...

set "STARTUP_FOLDER=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "CURRENT_DIR=%~dp0"

:: Create a shortcut in the startup folder
powershell -Command "$WScript = New-Object -ComObject WScript.Shell; $Shortcut = $WScript.CreateShortcut('%STARTUP_FOLDER%\Unreal Free Assets.lnk'); $Shortcut.TargetPath = '%CURRENT_DIR%unreal-free-assets.exe'; $Shortcut.WorkingDirectory = '%CURRENT_DIR%'; $Shortcut.Description = 'Monitors FAB/Unreal Marketplace for free assets'; $Shortcut.Save()"

if %ERRORLEVEL% EQU 0 (
    echo Successfully added to startup!
    echo The app will start automatically when Windows starts.
) else (
    echo Failed to add to startup.
)
pause
