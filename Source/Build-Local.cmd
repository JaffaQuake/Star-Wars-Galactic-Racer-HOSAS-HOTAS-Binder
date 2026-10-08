@echo off
setlocal
cd /d "%~dp0"

echo Building Galactic Racer HOSAS / HOTAS Bridge v1.8...
go build -ldflags="-H windowsgui" -o GalacticRacerHOSAS.exe .
if errorlevel 1 (
  echo.
  echo Build failed. Fix the Go errors above and try again.
  pause
  exit /b 1
)

rem The app loads ViGEmClient.dll from the same directory as the EXE.
if not exist "ViGEmClient.dll" (
  if exist "%LOCALAPPDATA%\GalacticRacerHOSAS\ViGEmClient.dll" (
    copy /Y "%LOCALAPPDATA%\GalacticRacerHOSAS\ViGEmClient.dll" "ViGEmClient.dll" >nul
    echo Copied ViGEmClient.dll from your installed app.
  ) else (
    echo.
    echo WARNING: ViGEmClient.dll was not found.
    echo The app will run, but virtual Xbox output will not work from this folder.
    echo Run the packaged Install.cmd once, or copy ViGEmClient.dll beside this EXE.
  )
)

echo.
echo Build complete:
echo   %CD%\GalacticRacerHOSAS.exe
pause
