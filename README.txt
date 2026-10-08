GALACTIC RACER HOSAS / HOTAS BRIDGE - v1.8
by JaffaQuake and Mars
==========================================

WHAT'S NEW IN v1.8
------------------
1. OPEN-SOURCE RELEASE PACKAGING
   The project is now distributed under the GNU General Public License v3.0 or later.
   The release package includes the full license, third-party notices, matching source code,
   development/build instructions, and a quick-start guide.

2. THIRD-PARTY LICENSE NOTICES
   THIRD_PARTY_NOTICES.txt documents the licenses for ViGEmClient, ViGEmBus, and the Go runtime.
   Those third-party components remain under their own licenses.

3. SOURCE LICENSE HEADER
   main.go now contains an SPDX GPL license identifier and copyright notice. This does not change
   application behavior; it makes the licensing of the source explicit.

FUNCTIONALITY RETAINED FROM v1.7
--------------------------------
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
