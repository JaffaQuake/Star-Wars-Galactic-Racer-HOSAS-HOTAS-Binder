GALACTIC RACER HOSAS / HOTAS BRIDGE - v1.91
by JaffaQuake and Mars
==========================================

WHAT'S NEW IN v1.91
-------------------
1. INSTALLER RELIABILITY HOTFIX
   Fixed the clean-install ViGEmClient.dll lookup that could fail on an x64 path. The installer now
   validates the DLL architecture instead of assuming an exact folder name.

2. MORE FLEXIBLE WINDOWS PATHS
   Local AppData, Desktop, and Start Menu locations are resolved through Windows special folders
   instead of being constructed from hard-coded path patterns.

3. MORE FLEXIBLE VIGEM DETECTION
   ViGEmBus detection checks services, drivers, and PnP devices instead of relying on one exact
   service name. The runtime can also find ViGEmClient.dll in the installed app folder when a local
   development build does not have its own copy.

4. SAFER LOCAL BUILDS
   Build-Local.cmd now searches common Go installation locations when Go is not yet visible in PATH,
   and gives clearer dependency diagnostics.

FUNCTIONALITY RETAINED FROM v1.9
--------------------------------
- PER-STICK THROTTLE AXIS MODE FOR HOSAS
   Each HOSAS side now has its own "Throttle axis" checkbox. When enabled, that selected axis
   is treated as an absolute throttle lever with 0% = -1, 50% = neutral, and 100% = +1.
   This allows a throttle lever to substitute for either side of the two-input HOSAS driving model.

- OPEN-SOURCE RELEASE PACKAGING
   The project is now distributed under the GNU General Public License v3.0 or later.
   The release package includes the full license, third-party notices, matching source code,
   development/build instructions, and a quick-start guide.

- THIRD-PARTY LICENSE NOTICES
   THIRD_PARTY_NOTICES.txt documents the licenses for ViGEmClient, ViGEmBus, and the Go runtime.
   Those third-party components remain under their own licenses.

- SOURCE LICENSE HEADER
   main.go now contains an SPDX GPL license identifier and copyright notice. This does not change
   application behavior; it makes the licensing of the source explicit.

OTHER FUNCTIONALITY RETAINED
----------------------------
- HOSAS / HOTAS Control Mode selector.
- HOSAS differential steering and optional Hybrid Preserve Thrust.
- HOTAS throttle -> Xbox RT and flight-stick X/Y -> Xbox Left Stick.
- Drive / Steer calculated-output monitor plus raw stick monitors.
- 100 / 200 / 250 / 500 / 1000 Hz selectable polling rate, saved at runtime.
- Rudder Off / Fine Steering / Move Camera / Keyboard Keys modes.
- Selectable pedal keyboard keys and activation threshold.
- On-Foot Off / Thumbstick -> WASD / Buttons -> WASD modes.
- Learned Xbox A/B/X/Y, bumpers, Start/Back, stick clicks and D-pad mappings.
- Learned Look Left / Look Right / Look Back keyboard camera mappings.
- Live mapped-button highlighting while not actively mapping.
- XInput readback diagnostics while mapping.
- Virtual Xbox controller exists only between Start Mapping and Stop Mapping.
- Auto-save plus manual Save Settings.
- JaffaQuake + Mars app icon and title branding.

INSTALL
-------
1. Extract the ZIP.
2. Double-click Install.cmd.
3. Follow any Windows prompts.

Read QUICKSTART.txt for the fastest HOSAS/HOTAS setup.

SETTINGS
--------
Your controller configuration is stored at:
  %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json

LICENSE
-------
Galactic Racer HOSAS / HOTAS Bridge is licensed under the GNU General Public License v3.0 or later.
See LICENSE.txt for the full terms.

Third-party components retain their own licenses. See THIRD_PARTY_NOTICES.txt.

SOURCE / LEARNING
-----------------
The matching Go source is included in the Source folder.
For local compilation instructions, read:
  Source\README-DEVELOPMENT.txt
