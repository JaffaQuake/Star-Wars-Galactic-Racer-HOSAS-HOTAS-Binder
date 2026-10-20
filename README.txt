v2.2.3 OFFICIAL — RUDDER PEDALS MODE DROPDOWN LAYOUT FIX
=========================================================
- The Rudder Pedals Mode dropdown is moved left slightly, leaving a clear
  gap before the Refresh devices button. The Mode label also has tighter
  bounds so its text and dropdown do not collide.
- This is a UI-only correction to v2.2.2 beta. Existing controller
  behavior, five profiles, global profile cycling and Walking Mode 45–55%
  throttle-neutral zone, saved settings, and driver setup are unchanged.
- The app and installer are branded v2.2.3. Windows hardware and DPI
  verification remains for users with the physical controllers/game.

v2.2.2 BETA — GLOBAL PROFILE CYCLING AND WALKING THROTTLE DEADZONE
===================================================================
- New GLOBAL Cycle Profile Button (Learn/Clear) beneath Walking Toggle.
  Physical button or POV hat advances to the next included numbered profile;
  the cycle wraps from 5 to 1. Every fresh press advances only once.
- Five checkboxes immediately below Profiles [1]-[5] choose which slots to
  include in cycling. All are checked on first upgrade. Unchecking a slot
  only skips cycling; its settings stay saved and it remains directly clickable.
  If no slots are included, cycling does nothing.
- Both clicking a different profile and cycling during play now send a neutral
  Xbox report, release held keyboard keys, apply the new profile and continue
  using the SAME virtual Xbox connection. Walking Mode resets OFF. Native
  Windows gameplay and physical hardware verification are still needed.
- Walking Mode: all throttle-style axes (HOTAT, Traditional HOTAS, and checked
  HOSAS Throttle Axis) have a STATIONARY zone at 45-55% physical travel.
  Below 45% = backwards, above 55% = forwards, with output scaled from zero
  at each boundary to full movement at the axis end. This is independent of
  racing Brake Behavior, brake zones, and curve midpoints. Spring-centered
  HOSAS axes, Single Stick and independent wheel pedals retain their behavior.
- The global assignment and checkbox flags are in hosas_profiles.json;
  existing v2.2.1 profile snapshots migrate without losing assignments.
- Cycling and walking help/tooltips explain these behaviors in the GUI.

v2.2.1 BETA — FIVE SAVED CONFIGURATION PROFILES
================================================
A Profiles row above Control Mode offers five independent slots: [1] [2] [3] [4] [5].
Profile 1 starts with your current configuration. Selecting an unused profile
copies the current configuration once; afterwards changes to devices, mode,
buttons and curves save independently to that slot. The selected slot stays
pressed. Switching an active Xbox controller disconnects it for safety;
re-initiate Xbox Controller after selecting the new profile.
Configurations are saved in %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_profiles.json.
The original hosas_config.json still contains your active configuration.

v2.2.0 BETA — WALKING MODE
================================
- Replaces the On-Foot Controls/WASD section with two fields: Walking Mode
  ON/OFF software toggle and Walking Toggle Button Learn/Clear assignment.
- Reuses existing physical inputs for walking movement/look across all five
  control modes. HOTAT/HOTAS throttle axes use fixed physical 50% neutral;
  wheel pedals use independently pressed accelerator/brake.
- The walking toggle binding is saved, but active walking state is never
  persisted and resets on virtual Xbox controller deactivation.
- v2.1.9 HOTAT brake layout fix and installer preserved.
- Refer to QUICKSTART.txt for mapping, limitations and operation.

v2.2.0 beta CONTROL MODE QUICK REFERENCE
==================================
Five modes: Podracer Style (HOSAS), Podracer Style (HOTAT), Traditional HOTAS,
Single Stick, and Driving Wheel. See QUICKSTART.txt for step-by-step setup.

Input plots now represent RAW physical movement; vertical bars fill 0-100%.
The green Drive / Steer graph remains processed output. Existing config is
preserved by the installer. Xbox output requires an operational ViGEmBus driver.

GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.2.1 beta
=================================================

OFFICIAL v2.1.7 RELEASE
-----------------------
Driving Wheel is no longer labeled Experimental following user hardware
confirmation. The Hand Brake row no longer displays the Rudder Pedals-only
"Pedal device" caption. The device/axis/mode controls, controller processing,
saved configuration format and installer dependency logic are unchanged.

v2.1.3 introduces a revised curve engine for all analog-tuned input profiles:
- Smooth single-stage Response Curve and Weighting (Left/Center/Right slider).
- Optional Two Stage Curve with independent curve and weighting sliders.
- Default diagonal midpoint (% input equals % output); optional Custom Midpoint
  reveals independent X/Y controls (no presets).
- Continuous, monotonic two-stage joins with tested slope limiting.
- Global Full Output Point, deadzone, Test Travel and Live Input are retained.
- Curve 1 is blue and Curve 2 is orangey yellow in the preview.
- Older non-linear calibration curves load identically until their weighting is
  deliberately edited. Digital-output/deadzone-only calibration remains simple.

EARLIER RELEASE: v2.1.2 migrates ALL main-window dropdowns to native Windows ComboBoxes,
including device, axis, control mode, pedal/auxiliary, keyboard and on-foot
selectors. Polling Rate and Input Curves/Axis Tuning already used the native
selector style. The option order and saved device ID/GUID behavior are retained.

The three bottom-right controller buttons now align with the mapping/tuning
content's right edge. No controller math or persistent JSON keys were changed.

The installed app uses a single CHANGELOG.md for all release history.
Installer audit, SHA256 and validation reports are development/release files,
not installed application data. On upgrade, Install.ps1 removes only generated
legacy release artifact filenames matching its strict allowlist; it preserves
hosas_config.json, unknown files and any personal files.

NOTE: Native popup lists may not appear in window-only screen recordings.
Full desktop capture is recommended if documenting open dropdown lists.

FEATURES
--------
- Podracer Style (dual-stick hybrid thrust), Traditional HOTAS and
  Driving Wheel mode.
- Wheel: centered steering; configurable device/axis/invert/Detect.
- Wheel Pedal 1 (Throttle): independent lever acceleration => virtual Xbox RT.
- Wheel Pedal 2: independent lever braking => virtual Xbox LT.
- Wheel Pedal 3 (Clutch): optional lever, Off/Boost/Drift.
- Hand Brake: separate lever device+axis, or digital button with Learn Btn.
  Hand Brake Mode is exclusively Drift (Xbox X) or Boost (Xbox A).
- Real native Windows right-panel tabs: Mappings (default) and embedded
  Input Curves/Axis Tuning (fixed dropdown/axis alignment).
- Calibration includes Polling Rate and Podracer-only Hybrid Preserve Thrust.
- Calibration values auto-apply; only Reset This Axis remains.
- Correctly named HOTAS and Wheel plot views, with dedicated diagnostics below plots.
- Colored red/blue/green circular live-monitor dots for the two physical inputs and combined Drive/Steer output.
- Widened Deactivate Xbox Controller button and added in-game Default controller scheme reminder.
- Configurable independent Axis Tuning/curves/deadzones for wheel/pedals,
  common Additional Inputs, and the established HOSAS/HOTAS modes.
- Additional Inputs: Centered supports Boost/Drift (negative X, positive A),
  Lever supports independent Boost or Drift. Digital action starts beyond deadzone.
- Historical v2.1.x: On-Foot Thumbstick->WASD previously displayed the saved thumbstick name. In v2.2.0 beta this is replaced by Walking Mode.
- Fix for disappearing Additional Input 1/2 text while device dropdowns open.
- Preserves 10 Hz visual refresh, mapping rows, WinMM helper isolation,
  hover-help, existing user configs and installed app paths.

INSTALL
-------
1. Extract this entire ZIP on a Windows x64 PC.
2. Run Install.cmd and follow prompts.
3. Existing configuration at %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json
   is NOT deleted by the installer.
4. Review QUICKSTART.txt, CHANGELOG.md and release notices.

Virtual Xbox output needs ViGEmBus / ViGEmClient; see installer and notices.
Windows GUI scaling and actual controllers have not been tested in this environment.


v2.1.4 calibration update: In Two Stage mode, normal Midpoint Position is now
a 10–90% slider with a live percentage readout. Enabling Custom Midpoint
hides the diagonal slider and (in v2.1.4) showed midpoint presets
alongside independent X Input / Y Output sliders. Presets are removed in v2.1.5. Calibration saves automatically;
curve calculations and saved configuration keys are unchanged.


v2.1.5 calibration layout: Two Stage now lists Curve 1, Curve 2, then
Midpoint. The Custom Midpoint checkbox precedes its controls; the normal
10–90% diagonal slider swaps in-place for separate X/Y sliders when checked.
The midpoint preset selector has been removed. All settings and curve math
are unchanged.


v2.1.6 calibration layout: Full Output Point now sits directly below
Deadzone on eligible analog axes. Removed the redundant standalone
Midpoint heading; the Custom Midpoint checkbox directly precedes its
mutually exclusive diagonal / custom X-Y slider controls. Curve math,
input processing and saved settings are unchanged.
