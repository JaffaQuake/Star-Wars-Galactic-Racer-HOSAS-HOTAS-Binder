GALACTIC RACER HOSAS / HOTAS BRIDGE - LOCAL DEVELOPMENT (v1.92 Beta)

The application is written in Go and uses native Windows APIs directly.

QUICK BUILD
1. Install Go for Windows.
2. Edit main.go and save it.
3. Close any running GalacticRacerHOSAS.exe.
4. Double-click Build-Local.cmd.
5. Run the newly built GalacticRacerHOSAS.exe in this Source folder.

Build-Local.cmd searches PATH and common Go install locations, builds a Windows GUI executable,
and copies ViGEmClient.dll beside the dev EXE when it can find the installed/release copy.

v1.92 BETA INPUT ARCHITECTURE
Physical flight axes can use:
  Auto -> DirectInput first, compatible WinMM fallback when the selected DirectInput device can be
          matched to its WinMM name.
  DirectInput -> broader X/Y/Z/Rx/Ry/Rz/Slider 1/Slider 2 path.
  WinMM -> legacy six-axis compatibility path.

Learned physical buttons/hats remain on WinMM in this beta.

HidHide integration is optional. The app looks for the official HidHideCLI.exe and
HidHideClient.exe under Program Files. It can register this executable on the allowlist and open
HidHide Configuration Client, but it deliberately does not auto-hide devices.

The UI remains a fixed-position Win32 layout inside a resizable scrollable window.
