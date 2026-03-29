@echo off
setlocal
cd /d "%~dp0"
echo.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\dev-windows.ps1" %*
set "ERR=%ERRORLEVEL%"
echo.
if not "%ERR%"=="0" (
  echo Script failed with exit code %ERR%.
)
pause
endlocal
