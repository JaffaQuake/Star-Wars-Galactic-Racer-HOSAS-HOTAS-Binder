# GalacticRacerHOSAS — Changelog

This consolidated changelog supersedes individually installed `RELEASE_NOTES_v*.txt` files.
Earlier entries below reproduce the supplied historical release notes for their corresponding
versions; they are not a claim that every historical feature has been tested on current hardware.
The newest version is described first. Controller configuration remains in
`%LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json`.

## v2.1.7 — Official Release / Driving Wheel Label Polish

- Promoted Driving Wheel out of Experimental status following user confirmation
  of working wheel controls. Updated the mode dropdown, hover help, live monitor,
  quickstart and current release documentation.
- In Driving Wheel mode, hide the stale "Pedal device" label beneath the
  Hand Brake heading. Restore that label for Podracer/HOTAS Rudder Pedals.
  The Hand Brake device selector, analog inputs and digital Learn Btn remain.
- Versioned the Windows GUI, installer and local build helper as v2.1.7.
- No changes to control mathematics, mappings, settings schema, user configuration,
  or ViGEmBus/ViGEmClient installation. Windows GUI/DPI and physical hardware
  regression checks for this exact build are still needed.

## v2.1.6 beta — Axis Tuning Layout Polish

- Moved global Full Output Point immediately beneath Deadzone on every eligible analog-output tuning page, including single- and two-stage curves.
- Removed the redundant standalone “Midpoint” heading from Two Stage controls; the Custom Midpoint checkbox now follows Curve 2 directly and is immediately followed by the standard diagonal midpoint slider or custom X/Y sliders.
- Reflowed tuning graph and footer to occupy the same height in normal/custom modes without gaps, preserving the parent tab boundaries.
- No changes to curve math, digital-only deadzone visibility, existing device inputs, saved settings, or installer cleanup behavior. Windows visual and hardware confirmation remain pending.

## v2.1.5 beta — Two Stage Layout Refinement

- In Input Curves / Axis Tuning, reordered Two Stage controls to Curve 1, Curve 2, then Midpoint.
- Moved the Custom Midpoint checkbox above the midpoint slider area. When unchecked, the normal 10–90% diagonal Midpoint Position slider sits directly below it; when checked, independent X Input and Y Output sliders take its place without leaving an empty row.
- Removed Left/Center/Right midpoint presets and their UI selection handler. Existing saved custom X/Y and default diagonal midpoint values load unchanged.
- Adjusted the graph height for normal/custom modes to maintain a consistent footer position inside the native tab and corrected spacing between descriptions, test travel, live input and Reset controls. Preserved curve math, colors, Full Output Point, live preview and deferred auto-save.
- Windows x64 build and portable/source regression tests performed; native Windows GUI and physical hardware acceptance tests still needed.

## v2.1.4 beta — Midpoint Slider UI

- Replaced the Two Stage Curve's normal numeric Midpoint Position field with a native slider from 10% to 90% and a live percentage readout. The join stays on the linear diagonal (X=Y).
- When Custom Midpoint is checked, the default Midpoint Position slider, label, and percentage display are hidden; existing X/Y sliders and midpoint presets are shown instead. Switching back restores the single slider.
- Maintained automatic application and deferred persistence of edits, established response-curve mathematics, blue/orange graph segments, global Full Output Point, and current settings schema.
- Windows native UI and physical hardware behavior still require validation on Windows.

## v2.1.3 beta — Single and Two Stage Response Curves

- Added smooth weighted curve shaping to all analog-output axis-tuning profiles
  across Podracer, Traditional HOTAS, Driving Wheel, and active auxiliary axes.
  Podracer's raw Left/Right axes intentionally retain their deadzone-only design;
  digital-only and inactive inputs continue to hide unnecessary curve controls.
- Single Curve includes Response Curve (0.20–3.00) and Weighting (Left–Center–Right,
  0–100). Curves use the approved weighted-power smooth-arc equation, without
  the previous simulator's artificially joined single-curve corner.
- Optional Two Stage Curve adds independent response strength and Weighting per
  stage. The regular Midpoint Position % travels on the linear diagonal X=Y.
  Custom Midpoint reveals Left/Center/Right presets and separate X/Y controls.
- Both stages meet continuously with matching tangents; monotonic Hermite
  limits prevent reversed output for extreme user-configured settings.
- The graph paints Curve 1 in blue and Curve 2 in warm orangey yellow.
- Global Full Output Point, deadzones, live calibration input, manual test travel,
  auto-save, existing input modes and virtual Xbox mappings remain supported.
- Upgrade compatibility: old non-linear response values keep the exact original
  exponent curve until the user adjusts Weighting; new tuning profiles default
  to centered weighting, linear strength and Two Stage Curve disabled.
- Host/race tests, Windows x64 GUI/helper build, Windows-test cross-compilation,
  static analysis and package checks completed. Windows real-device/UI testing
  still requires user verification.

## v2.1.2 beta — Native selector migration and release housekeeping

- Standardized all main-window selectors (control mode, input API, devices, axes,
  additional analogue modes, pedal modes, on-foot modes, key assignments, polling rate)
  on native Windows ComboBox dropdown lists. The existing option ordering, device IDs,
  DirectInput GUIDs, saved choices, conditional fields and mapping behavior are retained.
- Moved Test Xbox A, Initiate/Deactivate Xbox Controller and Save Settings together
  to right-align the button group with the right-side content area.
- Shipped one consolidated `CHANGELOG.md` instead of copying build-validation,
  installer-audit, versioned release notes and SHA256 artifacts into the installed app.
- Installer cleanup targets *only* known generated legacy release artifact filename
  patterns in its own installation folder. Unknown files, user mappings and settings are
  not touched.
- Note: native dropdown popups may not be captured by some window-only recording tools;
  full-desktop capture is recommended when documenting open dropdowns.

## Historical release notes

### v2.1.1-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.1.1 beta
==================================================
Source baseline: v2.1.0 beta Windows ZIP.

- Replaced Polling Rate's in-window listbox with a native COMBOBOX dropdown
  so the full five options (100/200/250/500/1000 Hz) can display above the
  embedded calibration child. Selected rate continues to use polling_hz.
- Only assigned and active inputs appear in Input to Configure. Combined
  Podracer outputs remain available if at least one side is assigned.
- Digital-output axes (Boost, Drift, Boost/Drift, Keyboard Keys and Driving
  Wheel analog Hand Brake) expose Deadzone only: curve, Full Output, graph
  and Test Travel are hidden, without erasing dormant tuning values.
- Physical digital handbrake buttons do not appear in analog calibration.
- Added Live Input checkbox to analog tuning Test Travel. At 10 Hz, the
  selected assigned physical axis drives the response graph marker and
  test percentage. Centered negative inputs retain a negative readout.
  Uncheck to restore manual Test Travel. Preview values do not modify saved
  calibration or virtual output. Physical controller may remain uninitiated.
- Retained all Podracer, Traditional HOTAS, Driving Wheel, live monitor,
  custom mapping, installer and existing user-config features.

Install: Extract the complete ZIP, then run Install.cmd on Windows x64.
Existing settings at %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json
are not overwritten. On-Windows visual/device testing is still required.
```

### v2.1.0-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.1.0 beta
==================================================
Built from the v2.0.11 beta Windows source, with two UI-only updates:

1. Changed main-window grey from v2.0.11 #C8C8C8 to approved lighter
   Option A #D4D4D4 (RGB 212,212,212). Tab controls and panel backgrounds
   otherwise retain native Windows styling.
2. Updated red, blue and green Live Input Monitor dots with an actual full
   black circular ring. Two nested filled ellipses (outside black, inside
   colored) replace the prior dark GDI pen border, which could look like a
   one-sided shadow when clipped by the indicator window region.

Unchanged: Podracer/HOTAS/Driving Wheel controls, axis tuning, polling,
controller creation, mapping modes, monitor refresh rate/positions, native
right-hand tabs, white help panel, and persistent configuration schema.

Install: Extract the complete ZIP and run Install.cmd on Windows x64.
Existing settings at %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json
are not overwritten. Windows UI and physical controller behavior require
on-device validation; this release was cross-built on Linux.
```

### v2.0.11-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.11 beta
===================================================
Based directly on supplied v2.0.10 beta Windows source.

1. Main-window chrome uses the approved Option B neutral grey #C8C8C8
   (RGB 200,200,200), providing clearer contrast with lighter themed tabs.
2. Fixed inconsistent white rectangular patches behind STATIC labels in
   Mappings and Input Curves/Axis Tuning: label backgrounds now explicitly
   match their correct parent pane and render with transparent text backing.
3. Colored live monitor indicators: left input red (#D84A4A), right input
   blue (#347ED4), combined Drive/Steer output green (#289753).
   Dots are circular with a dark outline, keeping chart grid visible behind
   their rounded corners. Existing positioning and 10 Hz refresh unchanged.
4. Existing native Windows tabs, calibration auto-save, UI layout, padded
   white tooltip panel, input mode options and stored configuration intact.

Install: Extract the whole archive and run Install.cmd as before. The
installer does not overwrite hosas_config.json under LocalAppData.

Validated: portable Go tests, Windows x64 main and WinMM builds, Windows
unit-test cross-compilation, Windows go vet; PE64/GUI subsystem, ZIP CRC,
manifest and content verification. Native Windows rendering, DPI, controller
hardware, and actual Windows test execution were not available here.
```

### v2.0.10-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.10 beta
====================================================
Built from the v2.0.9 beta Windows source. UI polish release.

1. Main-window background is a very slightly darker neutral grey (#ECECEC),
   contrasting gently with native Windows tabs and controls.
2. Removed the now-redundant vertical line alongside the tab panel.
3. Embedded axis-tuning child is sized using the native tab's usable content
   rectangle (TCM_ADJUSTRECT) plus a conservative inner right margin, instead
   of the former too-wide 655px window which painted over the tab border.
4. The bottom contextual-help strip is now a padded white card with a subtle
   grey border and shadow. Text is inset by 12px horizontally and 10px from
   the top; the card remains below HOTAS diagnostics and above footer buttons.
5. Kept Windows' standard tabs, buttons, dropdowns and sliders, with original
   mapping and controller logic and user settings compatibility unchanged.

Unchanged: HOSAS Podracer, Traditional HOTAS, Driving Wheel (EXPERIMENTAL),
wheel pedals and handbrake, auxiliary analog and Boost/Drift controls, mappings,
on-foot Thumbstick->WASD saved label, polling/curve auto-save, direct input,
defensive isolated WinMM helper, virtual Xbox and default controller reminder.

VALIDATION: Windows x64 app and WinMM helper cross-compiled. Portable Go tests
passed. Windows tests cross-compiled. Windows go vet passed. Release ZIP CRC,
PE-x64 architecture, and integrity checks completed.

NOT TESTED: Windows-native GUI appearance, native tab border rendering and DPI
scaling, actual controller input, or physical-wheel behavior. Please verify
on a Windows PC; this remains an experimental beta.
```

### v2.0.9-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.9 beta
===================================================
Built directly from the supplied v2.0.8 beta Windows source.

UI fixes (no changes to controller axis calculations or saved mappings):
1. Right-side Mappings / Input Curves/Axis Tuning uses a native Windows tab control
   with distinct selected/unselected tabs and a shared framed content region.
2. Polling Rate dropdown no longer hides the embedded axis-tuning page. The
   custom overlay manager leaves container windows visible and raises the list.
3. Embedded axis controls, selector, sliders, graph and Reset button now use the
   same left margin as the Calibration heading.
4. The left-pane tooltip/help box has returned near the bottom of the window,
   below the HOTAS verbose diagnostics, with 30px nominal separation.
5. The original vertical left/right panel divider is visible on BOTH right tabs
   because it belongs to the shared parent layout.

Kept from v2.0.8: Driving Wheel (EXPERIMENTAL), calibrated HOTAS and Podracer
profiles, additional analog Boost/Drift mapping, saved On-Foot thumbstick label,
input monitor controls, live 10Hz plots, direct input + defensive WinMM helper,
installer behavior and settings configuration schema.

VALIDATION: Windows x64 app and WinMM helper cross-compiled with Go 1.23.2;
portable Go tests passed; Windows Go tests cross-compiled; Windows Go vet passed;
release ZIP contents, archive CRC and binary architecture checked.

NOT TESTED: Native Windows GUI appearance/tab Z-order, dropdown behavior on a
Windows desktop, Windows 125%/150% DPI, or real controller hardware. Please
verify these on Windows; this is still an experimental beta.
```

### v2.0.8-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.8 beta
====================================================

BASELINE
--------
This release is based directly on the user-supplied v2.0.7-beta Windows ZIP/source.
Existing Podracer, Traditional HOTAS, experimental Driving Wheel control math,
HidHide/WinMM device enumeration, configuration keys, button mappings, and
thumbstick-assignment persistence are retained.

USER INTERFACE
--------------
- Replaced the detached axis-tuning popup with two right-side tabs:
  Mappings (selected on every launch) and Input Curves/Axis Tuning.
- Right-side mapping rows are wider to show longer saved device/button names;
  tabs retain their selection while switching control modes.
- Calibration at the top of the tuning tab now contains Polling Rate and the
  Podracer-only Hybrid Preserve Thrust toggle. Removed left-side calibration UI.
- The tuning page retains deadzone/response curve/full output controls,
  test-travel graph, and Reset This Axis. Removed Apply and Close.
- Tuning edits change the live output immediately; disk writes are coalesced
  for ~450 ms after a change and flushed on tab change or app exit.
- Reclaimed left-pane vertical space: On-Foot Controls and three live monitors
  moved up, with verbose HOTAS DirectInput diagnostic text below the charts.
- Mode-aware live plots: Podracer Left / Right / Drive-Steer; HOTAS
  Throttle-Brake / Stick / Drive-Steer; Driving Wheel
  Throttle-Brake / Wheel / Drive-Steer. On the first wheel plot positive is
  throttle, negative brake; wheel steering plots horizontally in the second.
- Driving Wheel's Pedal 1 label and axis-tuning name now say Throttle.
- Increased main default width to accommodate the tuning-page dimensions.
- Widened Initiate/Deactivate Xbox Controller button without moving the
  Test Xbox A footer action. Status on activation now reminds users that
  the in-game Galactic Racer controller scheme must be Default.

PRESERVATION / TEST LIMITATIONS
------------------------------
- Existing `%LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json` is preserved.
- No intentional changes to physical input polling, virtual Xbox output
  mappings, keyboard outputs, installer dependency resolution, or WinMM helper.
- Windows x64 build and Windows test suite cross-compile on Linux; portable
  Go unit tests pass. Windows native UI, scrolling, 100/125/150% DPI, device
  compatibility and gameplay still require hands-on validation.
```

### v2.0.7-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.7 beta
====================================================

SOURCE BASELINE
---------------
The v2.0.7 source was modified from the uploaded v2.0.6-beta ZIP/source,
which itself intentionally derives its layout from v2.0.4 rather than the
regressed v2.0.5. Existing normal-mode controls, mapping rows, help strip,
and footer button coordinates remain unchanged.

BUG FIXES
---------
- Additional Input 1/2 label rectangles no longer overlap the device
  dropdown list at x=150. Each label now ends at x=150, not x=155.
- On-Foot Controls shows "Mapped Thumbstick: <saved device> | Horizontal:
  <axis> | Forward: <axis>" when Thumbstick -> WASD is selected. The label
  is based on saved thumbstick assignment, not the most-active monitor.

NEW — DRIVING WHEEL (EXPERIMENTAL)
----------------------------------
- New, independent third Control Mode; saves its settings in driving_wheel.
- Wheel steering uses a centered -1..+1 axis, Device, Axis, Invert, Detect.
- Pedal 1 and Pedal 2 are fixed one-way lever accelerator/forward RT and
  brake/LT respectively. They have independent Device/Axis/Invert/Detect.
- Optional Pedal 3 (Clutch) is a lever with Off, Boost (A), Drift (X) Mode.
- Rudder Pedals is displayed as Hand Brake only in Driving Wheel mode.
  Its analog axis uses a lever (0..100%), or a learned DirectInput/WinMM
  digital button sends 100%. Hand Brake Mode choices are Boost or Drift.
- Each wheel input has its own axis tuning profile and saved assignment.
- Added Boost/Drift to centered Additional Analogue Inputs, and separate
  Boost, Drift selections to lever Additional Analogue Inputs. Traditional
  centered rudder mode has combined Boost/Drift as an option.
- Centered negative -> Drift (Xbox X), positive -> Boost (Xbox A).
  Invert swaps polarity. Post-deadzone motion triggers full digital output.

COMPATIBILITY
-------------
- No changes to old user setting keys; existing Podracer/HOTAS, on-foot,
  learned bindings and tuning survive updates. A new driving_wheel key was
  added without overwriting the original rudder settings.
- Main app title version upgraded to v2.0.7 beta. WinMM helper unchanged.
- Installer leaves %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json intact.

VALIDATION
----------
- Windows AMD64 GUI and WinMM helper compiled, both PE64 files.
- Portable axis/visibility/wheel-math tests ran and passed.
- Windows test suite cross-compiled, but was NOT executed on Windows.
- go vet passed with the legacy unsafe-pointer analyzer disabled; the
  baseline has existing Win32/DirectInput unsafe.Pointer analyzer warnings.
- Hardware detection, GUI DPI/scrolling, physical wheel/button mapping and
  the full Windows test suite require native Windows tests before broad release.
```

### v2.0.6-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.6 beta
====================================================

REBUILT FROM THE USER-SUPPLIED v2.0.4 BETA BASELINE
-------------------------------------------------
- Does not incorporate the v2.0.5 cosmetic layout changes.
- The original 54 tooltip registrations and in-window help strip are kept.
- Mapping rows and all footer buttons remain at their v2.0.4 coordinates,
  avoiding the prior overlap between tooltip help and controller actions.

COSMETIC CHANGES
----------------
- Header now reads: Flight controls -> Xbox Controls Mapping Bridge for
  Galactic Racer     Designed by JaffaQuake (Andy N.)
  It uses modestly larger bold typography, leaving all section fonts unchanged.
- Input API, Refresh Devices and HidHide Setup moved closer to Control Mode.
- Detect Left / Detect Right moved next to their Throttle axis checkboxes.
- Twist Axis heading renamed Additional Analogue Inputs; hardware rows and
  Axis Tuning menu renamed Additional Input 1 / Additional Input 2.
  Underlying saved profile keys, device selections, invert and output
  settings are unchanged.
- Section titles use Title Case at the original font size and weight:
  Input Devices, Additional Analogue Inputs, Rudder Pedals, Calibration,
  On-Foot Controls, Live Input Monitor, Standard Xbox Mappings and
  Galactic Racer Specific Mappings.
- Updates/socials, Initiate Xbox Controller and Save Settings button text
  made modestly larger and bold. Test Xbox A enlarged but not bold.
- Monitor during gameplay checkbox moved farther to the right.
- Scrollable content extents now calculated from actual control rectangles
  with 20px right/bottom padding instead of inflated fixed dimensions.
  Main window launches taller to accommodate unchanged footer coordinates;
  scrolling remains available on smaller displays.

VALIDATION
----------
- Go portable tests and static checks passed.
- Windows amd64 main GUI and isolated WinMM helper compiled.
- Windows-targeted tests cross-compiled (not executed here).
- Original 54 hover-help registrations, help text strip geometry, 43px
  mapping rows and bottom-action coordinates verified against v2.0.4.
- Installation paths and versioned docs checked; package SHA256 verified.
- Native Windows display-scaling, hover-tooltip and hardware testing pending.
```

### v2.0.5-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.5 beta
=====================================================

MAIN WINDOW COSMETIC REFINEMENTS
--------------------------------
- Replaced the opening description with a prominent, bold heading:
  "Flight controls -> Xbox Controls Mapping Bridge for Galactic Racer",
  followed on the same row by "Designed by JaffaQuake (Andy N.)".
- Closed the gap between Control Mode and Input API/Refresh Devices.
- Detect Left and Detect Right now sit closer to the Throttle Axis checkboxes,
  separated from the independent Control Mode/Input API settings at right.
- Twist Axis renamed to Additional Analogue Inputs, with left/right device
  labels renamed Additional Input 1 / Additional Input 2, including their
  Axis Tuning menu entries. This is strictly a display-name change; saved
  hardware bindings, axis profiles and JSON keys remain unchanged.
- Section headings use bold Title Case: Input Devices, Additional Analogue
  Inputs, Rudder Pedals, Calibration, On-Foot Controls, Live Input Monitor,
  Standard Xbox Mappings, and Galactic Racer Specific Mappings.
- Increased font size and bolded the Socials, Initiate Xbox Controller and
  Save Settings buttons. Test Xbox A is slightly larger but stays normal weight.
- Moved Monitor during gameplay to the right of the Live Input Monitor heading.
- Tightened mapping-row spacing while preserving double-line learned device
  text. Footer buttons now sit closer to the last mapping row.
- Scrollable canvas dimensions now use actual control bounding rectangles with
  20px right/bottom padding rather than oversized hard-coded widths/heights.
  Default window height was adjusted; scrolling is retained on small displays.

COMPATIBILITY
-------------
- Settings remain in %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json.
- All controller polling/monitoring, curve math, Invert/Output Negative,
  learned bindings and WinMM helper isolation are unchanged from v2.0.4.
- Existing settings, including polling_hz, are retained by the installer.

VALIDATION
----------
- Go tests and vet, Windows amd64 GUI/helper cross-compilation passed.
- Windows-targeted test binary compiled; source layout and integrity validated.
- ZIP checksums, installer references and PE subsystems checked.
- Native Windows visual and controller-hardware testing remains pending.
```

### v2.0.4-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.4 beta
=================================================

LAYOUT AND MAIN WINDOW
----------------------
- Calibration replaces the Driving header and retains Input Curves/Axis Tuning
  and Hybrid Preserve Thrust.
- Polling rate moved from the upper-right header to Calibration, to the left
  of the Axis Tuning button, with a wider rate selector.
- Removed redundant on-screen Forward steering note; tooltips remain.
- Links to new updates/socials relocated to the bottom-left of the main window,
  and still opens https://linktr.ee/jaffaquake in the default browser.
- Bottom-right action order: Test Xbox A (left), Initiate/Deactivate Xbox
  Controller (middle), Save Settings (right). Test Xbox A remains hidden
  unless the virtual controller is active.

OPT-IN GAMEPLAY MONITOR
----------------------
- LIVE INPUT MONITOR now includes a Monitor during gameplay checkbox, default
  OFF for fresh and existing configurations; saved in monitor_during_gameplay.
- When ON, monitor dots for Left Input, Right Input and Drive/Steer, mapping
  button tiles and on-foot mapped button indicators continue updating during
  gameplay; when OFF the existing pause behavior is preserved.
- The UI redraw timer stays at 10 Hz independently of controller polling,
  and the extra HOSAS horizontal input sampling is capped at 10 Hz while
  gameplay monitoring is on. A tooltip cautions about possible slight CPU use.
- This controls VISUAL feedback only; virtual Xbox output, mapping and
  controller polling retain their prior behavior.

COMPATIBILITY
-------------
- Settings remain in %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json.
- Existing mappings, tuning profiles, camera modes, device selection and
  polling frequency are unchanged. Historical configs load monitor Off.
- WinMM enumeration stays isolated in the helper executable.

VALIDATION
----------
- Go tests, vet and Windows x64 GUI/helper cross-compilation passed.
- Monitor pause/active policy tested; Windows-settings roundtrip cross-compiled.
- Release archive, executable subsystem, SHA256 and installer references checked.
- Native Windows visual and controller/hardware testing remains pending.
```

### v2.0.3-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.3 beta
=================================================

DEVICE-CONTEXTUAL SETTINGS
--------------------------
- Settings beside physical device dropdowns are now hidden until a real input
  device has been selected. Applies to Podracer Left/Right Inputs, HOTAS
  Throttle/Stick, Rudder Pedals, and both Twist Axes.
- Hidden settings include axis selectors, inversion, axis type, use/mode,
  throttle/brake controls, and relevant auxiliary mode-specific options.
- Existing configured device settings remain saved; hiding them does not reset
  their values. Device selectors and HOSAS Detect buttons remain accessible.
- Mode switching, hardware rescans, detected joystick configuration, and
  device dropdown changes refresh the visibility immediately.
- A "None" selection is available for Podracer inputs in WinMM mode as well
  as DirectInput, consistent with all other device selectors.
- Camera Left/Right tests require an actually selected pedal/twist in Move
  Camera mode and an active virtual Xbox controller.

HEADER / HIDHIDE STATUS
-----------------------
- "by JaffaQuake and Mars" credit moved down 5 logical pixels to align with
  the Flight controls line.
- Removed the static "HidHide: not installed" status indicator because the
  previous presence check only inspected the default Program Files path and
  could falsely report custom-drive installations as missing.
- Optional HidHide Setup button remains available. Its tooltip clarifies the
  default path limitation, and setup offers the official release page when
  its tools are not found there.

PRESERVED
---------
- Axis tuning/curves, polling-rate persistence, mappings, input semantics,
  hybrid thrust, and Windows compatibility WinMM isolation are unchanged.
- Installer preserves %LOCALAPPDATA%/GalacticRacerHOSAS/hosas_config.json.

VALIDATION
----------
- Portable Go tests, including device selection gating, pass.
- Go vet and cross-compilation of main Windows GUI/helper and Windows tests pass.
- ZIP, installer version references, PE subsystem and SHA256 checks pass.
- Physical Windows UI and actual hardware behavior remain to be validated.
```

### v2.0.2-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.2 beta
=================================================

UI COSMETIC / LAYOUT CHANGES
----------------------------
- Socials/updates button now sits horizontally beside the "by JaffaQuake
  and Mars" credit, directly above Control Mode. It starts at the Control
  Mode label's left edge and ends at the selector's right edge.
- The right-sidebar divider now stops at the bottom of the final Galactic
  Racer Specific Mapping row instead of continuing past the action buttons.
- Save settings is the leftmost button in the bottom-right controller action
  group, followed by Initiate/Deactivate Xbox Controller and Test Xbox A.

CONTEXT-AWARE TEST CONTROLS
---------------------------
- Test Xbox A is hidden until a virtual Xbox controller has been initialized
  and activated. Deactivation or initialization failure hides it again.
- Camera Left and Camera Right still require at least one Twist Axis or
  Rudder Pedals mode set to Move Camera. They now ALSO require an actively
  initialized virtual Xbox controller; otherwise the buttons are hidden.
- UI visibility refreshes after controller initialization, failed start,
  and deactivation, as well as when camera mode selections are changed.

UNCHANGED
---------
- Modes, axis curves, tuning profiles, twist axes, learned button mappings,
  camera-mode configuration and their saved values are unchanged.
- Polling rate remains saved as polling_hz in the user's config across
  software updates. The installer leaves hosas_config.json untouched.
- Existing defensive WinMM helper and ViGEm controller lifecycle preserved.

VALIDATION
----------
- Portable Go tests, including tests for inactive/active camera visibility,
  pass; go vet passes.
- Main Windows amd64 GUI executable and isolated WinMM helper compile.
- Windows-specific tests cross-compile; real Windows GUI/controller tests
  were not available in this environment.
- ZIP contents, installer references, executable subsystem, and SHA256
  manifests checked.
```

### v2.0.1-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.1 beta
================================================

MAIN WINDOW LAYOUT
------------------
- All 16 Standard Xbox and 5 Galactic Racer-specific learn/clear mappings
  are now in a full-height right-hand sidebar, parallel to the device setup
  and live input monitor. Assignments retain two text lines for long VKB names.
- Moved Initiate/Deactivate Xbox Controller, Test Xbox A, and Save Settings
  to the bottom-right, beneath the mappings.
- Moved Camera Left / Camera Right test buttons under Rudder Pedals.
  They are shown only when Pedal Mode, Left Twist Axis Mode, or Right Twist
  Axis Mode is set to Move Camera; otherwise they are completely hidden.
- Moved Polling rate directly under Control Mode selection.
- Placed a "Links to new updates/socials" button directly below
  "by JaffaQuake and Mars". This uses the default Windows web browser to
  open https://linktr.ee/jaffaquake when clicked.
- The window now defaults to a wider layout, with horizontal/vertical scroll
  when the display or window is smaller than the full layout.

POLLING RATE SETTINGS
---------------------
- Polling rate continues to be stored as polling_hz in
  %LOCALAPPDATA%\GalacticRacerHOSAS\hosas_config.json.
- The selected rate is saved on a selector change and with Save Settings,
  and is loaded from the same file after upgrading. The installer preserves
  the existing user configuration file.
- Supported saved rates: 100, 200, 250, 500, or 1000 Hz.

PRESERVED FROM v2.0.0 BETA
-------------------------
- Podracer / HOTAS input naming and mode-specific Axis Tuning popup.
- Independent HOSAS input deadzones, output curves and Full Output Point.
- HOTAS throttle/braking, Twist Lever Negative Output and Invert.
- Hybrid Preserve Thrust default-On / one-time config migration.
- Independent Apply and Close controls in Axis Tuning.
- WinMM device enumeration isolation / defensive helper.

VALIDATION
----------
- Portable axis math and camera button visibility unit tests passed.
- Go vet passed; Windows x64 GUI/WinMM helper executables cross-compiled.
- Windows-specific config round-trip tests compile but could not be executed
  in this environment; Windows UI/real-controller acceptance tests remain.
- SHA-256 and archive integrity checks performed on packaged release.
```

### v2.0.0-beta

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v2.0.0 beta
================================================

UI / FLOW
---------
- Podracer main group header: Input Devices, Left Input, Right Input.
- HOTAS main group header: Input Devices, Throttle, Stick.
- Visual stick monitor headers also use Left Input and Right Input.
- Moved Driving heading and controls immediately below the Rudder Pedals
  section and before On-foot, shifting On-foot controls downward without overlap.
- Renamed/widened button to Input Curves/Axis Tuning.
- Expanded Hybrid Preserve Thrust tooltip with OFF mode's average-thrust
  calculation and an illustrative 100%/0% = 50% example.

AXIS TUNING
-----------
- Selector dynamically filters Podracer and HOTAS-specific profiles:
  Podracer: Left Input, Right Input, Thrust Output, Steering Output,
  Rudder Pedals, Left Twist Axis, Right Twist Axis.
  HOTAS: Stick X, Stick Y, Throttle, Rudder Pedals,
  Left Twist Axis, Right Twist Axis.
- If control mode changes while the window is open, selector updates to
  the new mode. Pending values remain isolated in the popup draft.
- Podracer Left/Right Input: independent deadzones; Curve and Full Output
  controls, labels and values completely hidden, and popup shrinks vertically.
- All other profiles retain full Response Curve / Full Output Point settings.
- Apply saves all currently edited profiles and keeps the window open.
- Close discards unapplied drafts. Already applied changes remain saved.
- Existing curve math and settings serialization unchanged.

DEFAULT / CONFIG MIGRATION
--------------------------
- Hybrid Preserve Thrust defaults ON for fresh installs.
- Existing configurations migrate to default ON once in v2.0.0-beta and
  receive a preserve_thrust_v2_initialized JSON marker. After migration,
  explicitly choosing OFF is respected on subsequent launches.
- Existing learned mappings, camera settings, pedal/twist modes and
  previously saved axis_tuning profiles are preserved.
- DirectInput and isolated WinMM helper remain unchanged.

VALIDATION
----------
Go host-side math and mode-filter tests, Windows x64 cross-compilation, and
package/hash integrity checks performed. Windows GUI/hardware testing
is still required.
```

### v1.99

```text
GALACTIC RACER HOSAS / HOTAS BRIDGE — v1.99
==========================================
One Axis Tuning popup for independent analog control profiles.

Interface:
- Driving > Axis Tuning... launches a dedicated Windows tuning window.
- Driving's former Deadzone % field is removed, as are Twist Axis DZ% controls.
- Native sliders for Deadzone (0–40%), Response Curve exponent (0.50–3.00),
  Full Output Point (50–100%), and test physical travel (0–100%).
- Graph shows the response curve and a test travel/output indicator.
- Axis selector: HOSAS Left/Right deadzones, HOSAS Thrust/Steering outputs,
  HOTAS X/Y/Throttle, Rudder Pedals, Left/Right Twist axes.
- HOSAS per-stick curve/Full Output controls are disabled by design.
- Reset This Axis and Cancel do not modify other axis settings.
- Apply & Close updates the live controller processing and saves the settings.

Response pipeline:
- Physical axis -> normalized physical input -> independent deadzone
  -> response gamma after deadzone -> optional early full-output saturation.
- HOSAS left/right joystick deadzones are independent. Combined thrust and
  steering profiles are applied after differential mixing. This prevents
  uneven per-stick response curves from causing involuntary turns.
- HOTAS stick X/Y have independent response profiles. HOTAS throttle tuning
  operates only on positive acceleration after existing brake zone/threshold
  logic, so Analog Brake Zone and Digital at Idle behavior stays intact.
- Pedal and Twist Axis curves are applied before mode-specific authority.
- Lever invert and Output Negative remain independent and functional.
- Camera-specific deadzone overrides remain in their mode-specific areas.

Compatibility:
- Legacy v1.98.3 shared deadzone is copied to existing HOSAS stick deadzones,
  HOTAS stick X/Y and pedal deadzone on first v1.99 load; twist deadzones are
  copied from their own previous values. New response curves default to
  Linear (1.00) and Full Output defaults to 100%; output behavior is unchanged.
- Settings stored in axis_tuning JSON map; legacy fields kept for downgrade.
- WinMM helper isolation, mappings, installer and saved config path preserved.

Validation: compiled x64 Windows GUI application/helper, ran host-safe math
unit tests, compiled Windows-targeted tests and verified release ZIP.
Actual Windows GUI/hardware behavior still requires user testing.
```

### v1.98.3

```text
v1.98.3 — Release Notes
=====================
- Added independent Output Negative checkboxes to the Left Twist Axis and Right
  Twist Axis settings. They appear only when Axis Type is Lever.
- Output Negative is off by default. Without it, a lever outputs 0..+100%;
  with it, a lever outputs 0..-100% after deadzone.
- Invert continues to reverse lever travel before applying the output sign.
  This supports normal, inverted, negative, and inverted-negative behavior.
- Fine Steering, Pitch, Move Camera, and Keyboard Keys use the resulting signed
  lever output; Centered axis behavior is unchanged.
- Older settings and learned bindings load without migration; each checkbox
  persists under its own twist configuration as output_negative.
- The v1.98.2 UI spacing and v1.98.1 display fixes are preserved.
- Windows executables cross-compiled and arithmetic/JSON regression tests passed.
  Native Windows UI and controller hardware still require user validation.
```

### v1.98.2

```text
v1.98.2 — Release Notes
=====================
- Shifted the Twist Axis section and both its configuration rows down by
  22 logical pixels, increasing separation from the HOSAS/HOTAS controls
  and reducing the gap above Rudder Pedals.
- Removed the redundant "Independent analog inputs" explanation next to
  the Twist Axis heading. Centered / Lever behavior remains in tooltips.
- Preserved the v1.98.1 assignment-label and live-monitor dot fixes.
- No changes to physical input normalization, learned mappings, persisted
  configuration, the rudder-pedal controls or output behavior.
```

### v1.98.1

```text
v1.98.1 — Release Notes
=====================
- UI hotfix: widened assignment text, added two-line labels, taller mapping rows,
  and multiline button captions, keeping readable VKB button IDs in both columns.
- Fixed the live Left Stick / Right Stick / Drive-Steer indicator squares by
  deriving their positions from shifted layout rectangles and scroll offsets.
- Restored appropriate content scrolling for expanded mapping rows.
- Added hover help with the full mapping assignment name.
- No change to mappings, inputs, button binding keys, or control output semantics.
```

### v1.98

```text
v1.98 — Release Notes
=====================
- Added independent Left Twist Axis and Right Twist Axis configuration above
  Rudder Pedals, with physical Device/Axis, Invert, independent Deadzone,
  mutually exclusive Centered/Lever axis types (Centered default), and modes
  Off / Fine steering / Move Camera / Pitch / Keyboard keys.
- Added per-twist fine-steer authority, camera sensitivity/deadzone,
  keyboard positive/negative key selectors and activation thresholds.
- Analog twist axes read from the selected DirectInput or WinMM backend and
  are merged with existing flight-control output; disconnected inputs are ignored.
- Expanded 3-line live input monitor to show both twist values when inactive.
- Moved Pitch Up/Down below standard Xbox mappings with the Look controls in
  the 'Galactic Racer Specific Mappings' section. Added corresponding Xbox
  stick-arrow labels to each row, including the keyboard-emulated Look rows.
- Two-column mapping UI now has increased horizontal spacing.
- Preserved v1.97 button assignment keys and legacy pedal behavior.
- Preserved WinMM compatibility isolation from v1.96.1.
- Existing hosas_config.json files load without required user migration.
- Windows x64 native compilation passed with Go 1.23.2; runtime hardware/
  UI compatibility still requires testing on a Windows PC with VKB controllers.
```

### v1.97

```text
v1.97 — Release Notes
=====================

NEW
- Added Pedal Use -> Pitch for proportional Xbox Left Stick Y output.
- Added learned Pitch Up and Pitch Down mappings for full digital Left Stick Y output.
- Added Shift, Left/Right Shift, Left/Right Control, Alt, Left/Right Alt, and Tab options to Pedal
  Keyboard Keys mode.

WINMM COMPATIBILITY
- Carries forward the v1.96.1 WinMM isolation work.
- Auto and DirectInput avoid legacy WinMM device-capability enumeration during normal startup.
- WinMM device enumeration is performed through GalacticRacerWinMMHelper.exe so a failing legacy
  capability query cannot directly corrupt the main application's heap.
- The helper is launched automatically. Users do not run it manually.

BEHAVIOR NOTES
- Pitch pedal mode uses the selected pedal axis after Pedal Invert and the normal driving deadzone.
- Positive pedal input sends Left Stick Up; negative sends Left Stick Down.
- Pitch Up/Pitch Down learned mappings override analog Left Stick Y while held. If both are held,
  Left Stick Y returns to neutral.
- In Traditional HOTAS mode, selecting Pedal -> Pitch intentionally replaces the flight-stick Y
  output with pedal pitch. Digital Pitch mappings also override the current Left Stick Y output.

COMPATIBILITY
- Existing v1.96 and earlier hosas_config.json files remain compatible.
- Existing WinMM learned bindings remain supported.
```

### v1.96.1-Test

```text
Galactic Racer HOSAS / HOTAS Bridge v1.96.1 WinMM Isolation Test

- Removed legacy WinMM joystick capability enumeration from normal Auto/DirectInput startup.
- Added GalacticRacerWinMMHelper.exe to isolate WinMM enumeration from the main app.
- Each legacy joystick slot is queried in a separate hidden helper process so a faulty
  WinMM driver/slot cannot crash GalacticRacerHOSAS.exe during enumeration.
- WinMM compatibility mode remains available and existing settings remain compatible.
- DirectInput remains the primary device path for Auto mode.
- No changes to v1.96 control math, scrolling, mapping layout, or ViGEm output behavior.
```

### v1.96

```text
v1.96 — Release Notes

MAPPING UX
- Added a Clear button next to every Learn button in the Xbox/button mapping section and the on-foot WASD button section.
- Clear immediately removes the saved binding, updates the row to Not assigned, and writes the change to hosas_config.json.

TRIGGER CLARITY
- Expanded the Right Trigger and Left Trigger help text.
- The learnable trigger rows are explicitly described as optional manual/redundant controls.
- Analog Podracer Style and Traditional HOTAS behavior already generate RT/LT automatically; users do not need to learn these rows for normal acceleration/braking.

LIVE INPUT MONITOR
- Moved the Traditional HOTAS DirectInput Rx/Ry/Rz/Slider 1/Slider 2 readout out of the cramped top configuration row and into the Live Input Monitor area.
- DirectInput diagnostics still pause while active Xbox output is running to keep UI overhead low.

RETAINED FROM v1.95
- Working mouse-wheel and scrollbar implementation.
- Correct DirectInput Rx/Ry/Rz/Slider binding.
- Podracer Style / Traditional HOTAS modes.
- Traditional HOTAS Off / Digital at Idle / Analog Brake Zone behavior.
- Backend-aware WinMM / DirectInput Learn behavior and legacy binding preservation.
- HidHide setup helper and ViGEm/XInput output lifecycle.

Compatibility note: physical hardware compatibility depends on the device/driver exposing standard WinMM or DirectInput controls.

FINAL RELEASE POLISH
- Increased horizontal spacing between the two button/switch mapping columns for clearer visual separation.
- No control-logic or compatibility behavior was changed from the final v1.96 pre-release build.
```

### v1.96-beta

```text
v1.96 Beta — Release Notes

MAPPING UX
- Added a Clear button next to every Learn button in the Xbox/button mapping section and the on-foot WASD button section.
- Clear immediately removes the saved binding, updates the row to Not assigned, and writes the change to hosas_config.json.

TRIGGER CLARITY
- Expanded the Right Trigger and Left Trigger help text.
- The learnable trigger rows are explicitly described as optional manual/redundant controls.
- Analog Podracer Style and Traditional HOTAS behavior already generate RT/LT automatically; users do not need to learn these rows for normal acceleration/braking.

LIVE INPUT MONITOR
- Moved the Traditional HOTAS DirectInput Rx/Ry/Rz/Slider 1/Slider 2 readout out of the cramped top configuration row and into the Live Input Monitor area.
- DirectInput diagnostics still pause while active Xbox output is running to keep UI overhead low.

RETAINED FROM v1.95
- Working mouse-wheel and scrollbar implementation.
- Correct DirectInput Rx/Ry/Rz/Slider binding.
- Podracer Style / Traditional HOTAS modes.
- Traditional HOTAS Off / Digital at Idle / Analog Brake Zone behavior.
- Backend-aware WinMM / DirectInput Learn behavior and legacy binding preservation.
- HidHide setup helper and ViGEm/XInput output lifecycle.

Beta note: physical hardware compatibility still depends on the device/driver exposing standard WinMM or DirectInput controls.
```

### v1.95-beta

```text
v1.95 Beta — Release Notes

CONTROL MODE / UI
- Renamed the control-mode choices to "Podracer Style" and "Traditional HOTAS" and refreshed their in-app help text.
- Existing saved internal mode values remain compatible (HOSAS/HOTAS are still accepted in configuration files).

DIRECTINPUT AXIS FIX
- Rebuilt the DirectInput data format so X/Y/Z, Rx/Ry/Rz, Slider 1 and Slider 2 bind to their actual DirectInput object GUIDs instead of generic axis slots.
- This targets devices where joy.cpl correctly shows an axis such as Z Rotation but the previous beta did not read that same axis when selected.
- Added a HOTAS DirectInput live readout for Rx, Ry, Rz, Slider 1 and Slider 2 on the selected stick/throttle device while Xbox output is inactive.

TRIGGER MAPPINGS
- Renamed the old manual Brake binding to "Left Trigger" and added a separate learnable "Right Trigger" binding.
- Left Trigger and Right Trigger are placed directly below the bumper mappings.
- Existing v1.94 BRAKE bindings migrate automatically to Left Trigger.
- Learned trigger mappings can be used independently of analog throttle logic.

TRADITIONAL HOTAS BRAKING
- Added Brake Behavior: Off / Digital at Idle / Analog Brake Zone.
- Digital at Idle retains the existing behavior and remains the upgrade default. Idle Brake Threshold % controls where LT snaps to 100%.
- Analog Brake Zone reserves the lower throttle percentage as a proportional LT range: minimum throttle = 100% LT, fading smoothly to 0% LT at the top of the brake zone.
- Above an Analog Brake Zone, the remaining throttle travel is rescaled to the full 0–100% RT range.
- Brake-specific fields are context-sensitive: only the relevant percentage field is shown for the selected brake mode.

SCROLLING FIX
- Added WM_MOUSEWHEEL support.
- Child controls are now repositioned in one batched DeferWindowPos transaction instead of being moved one-by-one.
- The full window and child controls are explicitly repainted after scroll changes to prevent stale lines, skipped pixels and overlap artifacts.
- Open custom selector lists are still closed before scrolling.

RETAINED
- Backend-aware WinMM / DirectInput Learn behavior and legacy binding preservation.
- Capture-friendly in-window selectors and in-window hover help.
- Resizable fixed-size UI canvas with scrollbars.
- HidHide setup helper, ViGEm lifecycle, polling-rate selection, Podracer hybrid-thrust logic, pedal modes and on-foot mappings.

Beta note: physical DirectInput/HOTAS behavior still needs testing across hardware. The build environment can validate compilation but cannot exercise real Windows controller devices.
```

### v1.94-beta

```text
v1.94 Beta — Release Notes

- Custom in-window selector lists now close when the user clicks outside the open list, including elsewhere inside the main application window.
- Selector focus loss also dismisses the open list without changing the selection.
- Added a new learnable Brake mapping directly below Look Back.
- Manual Brake sends Xbox Left Trigger (LT) at 100% while pressed and is intentionally independent of RT/forward thrust, allowing simultaneous throttle + brake input when desired.
- HOTAS Brake Threshold automatic brake-at-zero behavior is retained as a separate feature.
- Existing v1.93 bindings/configuration remain compatible; the new BRAKE binding starts unassigned.
- All v1.93 Beta 2 backend-aware learning, DirectInput/WinMM compatibility, resizable UI, HidHide helper, and virtual-controller lifecycle behavior are retained.

Beta note: the new manual LT behavior intentionally allows RT and LT to be active simultaneously when a user holds forward thrust and presses the learned Brake control. Game behavior for simultaneous triggers can vary by title.
```

### v1.93-beta

```text
v1.93 Beta 2 — Release Notes

- Learn buttons/hats now use the selected input backend.
- DirectInput learned bindings store device GUID + button/POV information; WinMM bindings remain unchanged.
- Auto learning watches both APIs and prefers DirectInput.
- DirectInput button data format expanded to 128 buttons plus four POV hats.
- Added HOTAS Brake Threshold entry (default 5%): throttle at/below threshold -> LT 100%, RT 0.
- Renamed Start Mapping to Initiate Xbox Controller / Deactivate Xbox Controller.
- Replaced external/top-most tooltip popups with an in-window help bar.
- Replaced native popup combo lists with capture-friendly in-window selector lists.
- Reworded learning prompts to be hardware-neutral.
- Existing configs and learned WinMM mappings are preserved.

Beta note: DirectInput button/POV learning and the new in-window selector UI require wider hardware testing.
WinMM remains available as the compatibility backend.


Beta 2 hotfix:
- Fixed capture-friendly selector lists being painted behind buttons/controls.
- Open selectors now force top child-window Z-order and temporarily hide only overlapping controls, restoring them immediately when the selector closes.
- No controller logic, bindings, backend behavior, or config format changed from v1.93 Beta.
```

### v1.92-beta

```text
v1.92 Beta — Release Notes

- Added Auto / DirectInput / WinMM axis input backends.
- Existing configuration files migrate to WinMM compatibility to avoid changing known-good setups.
- DirectInput adds Rx/Ry/Rz and Slider 1/2 to flight-axis selectors.
- Renamed control modes to HOSAS / Podracer Style and HOTAS / Traditional Steering.
- Added optional HidHide Setup helper for double-input troubleshooting.
- Made the Win32 window resizable/maximizable with fixed-size content and scroll bars when needed.
- Kept ViGEm/XInput output unchanged in this beta.
- Learned button/hat mappings continue to use WinMM for stability.

Beta note: DirectInput support is deliberately isolated to analog flight-axis input so users can
switch back to the existing WinMM path immediately if a particular device behaves unexpectedly.
```
