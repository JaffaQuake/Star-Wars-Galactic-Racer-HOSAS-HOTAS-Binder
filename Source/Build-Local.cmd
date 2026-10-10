@echo off
setlocal EnableExtensions
cd /d "%~dp0"

echo Building Galactic Racer HOSAS / HOTAS Bridge v1.92 Beta...

set "GOEXE="
for /f "delims=" %%G in ('where go.exe 2^>nul') do if not defined GOEXE set "GOEXE=%%G"
if not defined GOEXE if exist "%ProgramFiles%\Go\bin\go.exe" set "GOEXE=%ProgramFiles%\Go\bin\go.exe"
if not defined GOEXE if exist "%LOCALAPPDATA%\Programs\Go\bin\go.exe" set "GOEXE=%LOCALAPPDATA%\Programs\Go\bin\go.exe"

if not defined GOEXE (
  echo.
  echo ERROR: Go could not be found.
  echo Install Go, reopen your IDE/terminal, or add Go\bin to PATH.
  echo Common location: C:\Program Files\Go\bin\go.exe
  pause
  exit /b 1
)

echo Using Go: %GOEXE%
"%GOEXE%" build -ldflags="-H windowsgui" -o GalacticRacerHOSAS.exe .
if errorlevel 1 (
  echo.
  echo Build failed. Fix the Go errors above and try again.
  pause
  exit /b 1
)

rem The runtime checks beside this EXE first, then the installed app folder.
rem Copying the DLL locally keeps the dev build portable and avoids surprises.
if not exist "ViGEmClient.dll" (
  if exist "%LOCALAPPDATA%\GalacticRacerHOSAS\ViGEmClient.dll" (
    copy /Y "%LOCALAPPDATA%\GalacticRacerHOSAS\ViGEmClient.dll" "ViGEmClient.dll" >nul
    echo Copied ViGEmClient.dll from your installed app.
  ) else if exist "..\ViGEmClient.dll" (
    copy /Y "..\ViGEmClient.dll" "ViGEmClient.dll" >nul
    echo Copied ViGEmClient.dll from the release folder.
  ) else (
    echo.
    echo NOTE: No local ViGEmClient.dll was found.
    echo The v1.92 Beta runtime will also check the normal installed app folder when Start Mapping is pressed.
    echo If neither location has the DLL, virtual Xbox output will not start.
  )
)

echo.
echo Build complete:
echo   %CD%\GalacticRacerHOSAS.exe
pause
