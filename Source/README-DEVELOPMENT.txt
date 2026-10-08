GALACTIC RACER HOSAS / HOTAS BRIDGE - LOCAL DEVELOPMENT (v1.8)
=======================================================

The source is written in Go. main.go contains the current application.

FASTEST WAY TO BUILD
--------------------
Double-click Build-Local.cmd in this Source folder.

It runs:
  go build -ldflags="-H windowsgui" -o GalacticRacerHOSAS.exe .

It also tries to copy ViGEmClient.dll from the installed app folder into this Source folder.
That DLL must be beside the development EXE for virtual Xbox output to work.

WHY A SELF-BUILT EXE CAN LOSE XBOX OUTPUT
-----------------------------------------
The program dynamically loads:
  ViGEmClient.dll

from the same directory as the running GalacticRacerHOSAS.exe. A Go build creates only the EXE;
it does not automatically copy that DLL. If the DLL is missing, joystick/keyboard features can still
run, but Start Mapping cannot create the virtual Xbox controller.

The normal installer places the dependency in:
  %LOCALAPPDATA%\GalacticRacerHOSAS\ViGEmClient.dll

INPUT BACKENDS
--------------
The application deliberately uses several Windows input/output paths because each is suited to a
specific job:

  WinMM joystick API
    Main HOSAS forward/back axes, HOTAS throttle/stick axes, rudder, buttons and POV hats. This remains the stable driving path.

  DirectInput 8
    On-foot mini-stick discovery and reading in Thumbstick -> WASD mode. DirectInput provides a
    DIJOYSTATE-style view with X, Y, Z, Rx, Ry, Rz and two slider axes. This is the path intended to
    match X Rotation / Y Rotation shown by joy.cpl on VKB hardware.

  ViGEmClient / ViGEmBus
    Creates the virtual Xbox 360 controller only while Start Mapping is active.

  Win32 SendInput
    Sends keyboard outputs such as pedal camera keys, Look Left/Right/Back and on-foot WASD.


HOSAS / HOTAS CONTROL MODE
--------------------------
The saved config field control_mode selects HOSAS or HOTAS.

HOSAS uses the existing two-stick differential calculation.

HOTAS uses these saved WinMM selections:
  hotas_throttle_id / hotas_throttle_axis / hotas_throttle_invert
  hotas_stick_id / hotas_stick_x_axis / hotas_stick_y_axis
  hotas_stick_invert_x / hotas_stick_invert_y

The throttle axis is normalized from -1..+1 into Xbox RT 0..255 and never generates LT.
The flight-stick X/Y axes map directly to Xbox Left Stick X/Y after the normal stick deadzone.

DIRECTINPUT THUMBSTICK DETECTION
--------------------------------
Set On-Foot mode to Thumbstick -> WASD. The DirectInput monitor shows the eight axis values from the
most active DirectInput game-controller device while game mapping is stopped.

Detect Thumbstick is a two-stage calibration:
  1. Push the mini-stick RIGHT.
  2. Return to center and push it FORWARD.

The app records the DirectInput device GUID, axis index, center point and positive travel span for
each direction. Side and forward can be stored from different logical DirectInput devices.

ON-FOOT BUTTON MODE
-------------------
Set On-Foot mode to Buttons -> WASD to reveal four Learn controls. W/A/S/D can each be learned from
any WinMM button or POV/hat input supported by the existing Learn system.

POLLING RATE
------------
The controller polling rate is selectable directly in the application:
  100 / 200 / 250 / 500 / 1000 Hz

The selection is saved in hosas_config.json and the polling ticker changes at runtime. Windows
scheduling and the physical USB device report rate still determine how many genuinely new samples
are available.

CONTEXT-SENSITIVE UI
--------------------
Mode-specific controls are grouped and shown/hidden by updateContextVisibility(). If you add a new
mode-specific option, add its HWND to the appropriate control group and update that function.

SETTINGS
--------
Development and installed builds share the same settings file:
  %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json

APP ICON
--------
The app icon source is GalacticRacerHOSAS.ico in this Source folder. It is embedded into the Go
executable for the window/taskbar using go:embed. The packaged installer also places a copy beside
the installed EXE so Desktop and Start Menu shortcuts use the same icon.


LICENSE / CONTRIBUTIONS
-----------------------
This source is licensed under the GNU General Public License v3.0 or later.
See ..\LICENSE.txt and ..\THIRD_PARTY_NOTICES.txt.

If you distribute a modified binary, provide the corresponding source under the GPL terms and
preserve applicable third-party notices.
