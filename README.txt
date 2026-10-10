GALACTIC RACER HOSAS / HOTAS BRIDGE - v1.92 Beta
by JaffaQuake and Mars
===============================================

WHAT'S NEW IN v1.92 BETA
------------------------
1. INPUT API SELECTOR
   New Input API setting:
     - Auto (DirectInput first)
     - DirectInput
     - WinMM (compatibility)

   DirectInput exposes the broader flight-controller axis set:
     X / Y / Z / Rx / Ry / Rz / Slider 1 / Slider 2

   This is intended to improve compatibility with throttles and flight hardware that place their
   main lever on rotation or slider axes. Existing configs are migrated to WinMM so known-good
   setups keep their previous behavior until the user chooses another backend.

   In this beta, learned buttons and POV hats still use the proven WinMM button path.

2. HOSAS / PODRACER STYLE + HOTAS / TRADITIONAL STEERING
   The Control Mode labels now describe the actual driving style:
     - HOSAS / Podracer Style
     - HOTAS / Traditional Steering

   Hover the selector for a short explanation of each mode.

3. OPTIONAL HIDHIDE SETUP HELPER
   Added a HidHide Setup button for users experiencing double input.
   If HidHide is installed, the app attempts to add itself to HidHide's application allowlist and
   opens the official HidHide Configuration Client. The user still chooses which physical devices
   to hide; v1.92 Beta intentionally does not auto-hide hardware.

   HidHide is optional and is NOT bundled with this project.

4. RESIZABLE / SCROLLABLE WINDOW
   The main window can now be resized or maximized without stretching the existing buttons,
   monitors, or mapping tables. If the window is smaller than the fixed content area, horizontal
   and/or vertical scroll bars appear. Enlarging the window simply exposes more whitespace.

5. DIRECTINPUT AXIS NORMALIZATION
   v1.92 Beta asks DirectInput to normalize analog axes to a consistent signed range before our
   HOSAS/HOTAS calculations are applied. This keeps the control math independent from a device's
   native raw range where the driver supports DirectInput range properties.

RETAINED FUNCTIONALITY
----------------------
- HOSAS differential steering and optional Hybrid Preserve Thrust.
- Per-side HOSAS "Throttle axis" option.
- HOTAS throttle -> Xbox RT and flight-stick X/Y -> Xbox Left Stick.
- Rudder Off / Fine Steering / Move Camera / Keyboard Keys modes.
- On-Foot Off / Thumbstick -> WASD / Buttons -> WASD modes.
- Learned Xbox buttons, D-pad, native Look Left / Right / Back camera mappings.
- 100 / 200 / 250 / 500 / 1000 Hz selectable polling rate.
- Live raw-stick and Drive / Steer monitors when mapping is stopped.
- XInput readback diagnostics while mapping.
- Virtual Xbox controller lifecycle: created only by Start Mapping, removed by Stop Mapping.
- v1.91 installer reliability fixes and safer local build helper.

INSTALL
-------
1. Extract the ZIP.
2. Double-click Install.cmd.
3. Follow any Windows prompts.

Read QUICKSTART.txt for the fastest setup.

SETTINGS
--------
Your controller configuration is stored at:
  %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json

LICENSE
-------
Galactic Racer HOSAS / HOTAS Bridge is licensed under the GNU General Public License v3.0 or later.
See LICENSE.txt for the full terms. Third-party components retain their own licenses; see
THIRD_PARTY_NOTICES.txt.

SOURCE / LEARNING
-----------------
The matching Go source is included in the Source folder.
For local compilation instructions, read Source\README-DEVELOPMENT.txt.
