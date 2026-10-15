v2.1.7 OFFICIAL RELEASE NOTES
==============================
Based directly on the user-supplied v2.1.6 beta source and Windows ZIP.
UI-only update: hide the legacy "Pedal device" STATIC label in WHEEL mode,
restore it under Rudder Pedals in HOSAS/HOTAS, and remove Experimental from
current Driving Wheel labels, monitoring and docs. No output processing,
calibration math, config migration, or installer dependency behavior changed.
Previous beta notes below are historical. Physical Windows/UI acceptance
for the exact v2.1.7 binary remains to be performed.

v2.1.6 BETA UI CHANGE NOTES
==========================
Based directly on v2.1.5 beta source. No curve math or config schema changes.
Full Output Point has been moved to the row immediately beneath Deadzone,
on all analog output profiles. Removed only the standalone Midpoint heading;
the normal Midpoint Position label and the Custom Midpoint checkbox remain.
The graph heights for standard and custom two-stage modes are reflowed to a
shared bottom position (620px), keeping Reset inside the 807px tuning child.
Digital-only calibration remains deadzone-only. Windows UI/hardware verification
is required on native Windows.

v2.1.5 BETA UI CHANGE NOTES
==========================
Based directly on v2.1.4 beta source. No curve engine or settings schema edits.
The two-stage tab now shows Curve 1 -> Curve 2 -> Midpoint. In Midpoint,
Custom Midpoint checkbox is followed by either the default 10-90% diagonal
slider or separate X/Y sliders in the same location. Legacy midpoint preset
UI and its event handler have been removed. Graph preview heights reflow to
keep the footer within the 807px embedded page with both states.
Keep testing that hidden controls do not erase any saved profile values.

v2.1.5 BETA DEVELOPMENT NOTES
================================
Built from the user-confirmed v2.1.2 beta source. Smooth weighted-power
single-stage and monotone Hermite joined two-stage curves, with configurable
strength, weighting and midpoint. Respect every saved axis-tuning profile,
existing deadzone/Full Output Point and response pipeline. New profiles have
centered weighting; old non-linear profiles keep weight 0 until edited to
preserve exact behavior. A single shared global full-output point is used.
Two-stage graph preview: blue initial curve, warm orange/yellow second curve.
Expanded embedded tuning height to 807 inside native tab frame; compact custom
midpoint controls remain inside it. Portable math and serialization tests,
Windows cross-build and packaging checks are required before delivery.

v2.1.2 BETA DEVELOPMENT NOTES
==============================
Based directly on user-supplied v2.1.1 beta source. All main-window dropdowns
now use native COMBOBOX/CBS_DROPDOWNLIST, using the same option arrays, selected
indices, device ID/GUID mappings, conditional visibility and persistence.
The former custom popup LISTBOX is removed from the main-window UI; native
popup lists may not be captured by window-only recording tools.
Closed selector layout bounds remain 26px rather than the popup-list height.
The three controller action buttons are right-aligned with tab content while
keeping widths and gaps. Changelog consolidation and strict filename-only
installer cleanup keep development artifacts out of installed AppData.
Tests and validation are recorded outside the installed application folder.
Windows GUI, real device and DPI verification still requires a Windows PC.

v2.1.1 BETA DEVELOPMENT NOTES
================================
Based on supplied v2.1.0 source. Polling Rate now uses a native ComboBox popup
rather than the common in-client LISTBOX (which was behind tuning pane).
All five options still save to polling_hz. Source filtering of calibration
entries uses assigned/active device and mode, while leaving dormant tuning
profiles in the JSON. Digital-output axes show deadzone only; graph and Test
Travel are hidden. Live Input uses WM_TIMER at 10 Hz and does not save travel.
Virtual Xbox activation is not required for preview. Native Windows GUI and
actual physical-controller verification are pending.

v2.1.0 BETA DEVELOPMENT NOTES (HISTORICAL)
==============================================
Built directly from the v2.0.11 beta source. Presentation-only updates:
- Main window background now Option A #D4D4D4 (RGB 212,212,212).
- Monitor dots still show red / blue / green with 12px geometry. To make
  the circular black edge truly symmetric, WM_PAINT fills an outer black
  ellipse and then a smaller inset colored ellipse with NULL_PEN. Both fills
  are bounded by the existing elliptic window region, leaving chart lines
  unobscured at the four corners. No motion/input behavior was changed.
- All tab, tooltip, mapping, tuning, controller and persistence code stays.
- Windows screenshots and hardware still require native Windows testing.

v2.0.11 BETA DEVELOPMENT NOTES (HISTORICAL)
=============================================
Source base: v2.0.10 beta, Windows x64, UI-only refinements.
- User-selected outer-window grey #C8C8C8 (RGB 200,200,200) applies to
  mainBackgroundBrush and to the custom Help Panel shadow surroundings.
- WM_CTLCOLORSTATIC now explicitly sets TRANSPARENT text background mode,
  the current pane's matching background COLORREF and brush. For tab page
  labels, use COLOR_BTNFACE, and for the outer main panel use #C8C8C8.
  Embedded curve page handles its own static-label messages the same way.
- Live monitor markers are 12px custom-painted circular children with a
  dark outline; each owns an elliptical window region so no rectangular
  background patch covers the graph's grid/axes while the dot moves.
  Left=#D84A4A red; Right=#347ED4 blue; Drive/Steer=#289753 green.
- Monitor motion and 10Hz UI update policy unchanged, across all modes.
- All saved settings, tooltips, mappings, calibration and input code retained.
- Native Windows GUI/real-controller/display-scale verification is still needed.

v2.0.10 BETA DEVELOPMENT NOTES (HISTORICAL)
===============================
Source base: v2.0.9 beta. UI presentation and sizing only.
- Outer background brush: neutral grey RGB(236,236,236); native buttons,
  dropdowns, and SysTabControl32 retain their Windows theme.
- Shared free-standing vertical divider removed (native tab frame replaces it).
- Embedded axis window width derives from TCM_ADJUSTRECT, with 12px right
  inset; it must not cover the tab border as the old 655px child did.
- New capture-friendly GalacticRacerHelpPanel child uses white fill,
  subtle in-window shadow, a 1px grey border, and 12px text inset.
- Existing tooltip strings, data, control mappings, migration and installer
  schema are not changed. Verify fonts, borders and DPI on real Windows.

v2.0.9 BETA DEVELOPMENT NOTES (HISTORICAL)
==========================================
Base: v2.0.8 beta, Windows x64. UI-only bugfix release.
Native SysTabControl32 uses WM_NOTIFY/TCN_SELCHANGE. The shared panel
separator formerly lay outside the per-tab visibility groups (removed in v2.0.10). In-window custom
combobox dropdowns must NOT hide the tab or embedded tuning parent.
The help strip is at virtual y=940, below HOTAS diagnostics y=855..910.
Embedded axis tuning control x=0 is flush with the Calibration header.
Native Windows testing remains necessary for Z-order and DPI behavior.

GALACTIC RACER HOSAS / HOTAS BRIDGE - LOCAL DEVELOPMENT (v2.1.0-beta)
================================================================

The release includes matching Go source for both executables.

MAIN APP
--------
Source/main.go

WINMM HELPER
------------
Source/WinMMHelper/main.go

Build-Local.cmd builds both executables on Windows. The helper exists only to isolate legacy WinMM
device-capability enumeration from the main process. Users do not launch it directly.

v2.1.0-beta INPUT / OUTPUT NOTES (HISTORICAL)
--------------------------
- Auto and DirectInput use the DirectInput device list during normal startup.
- WinMM capability enumeration runs through GalacticRacerWinMMHelper.exe when required.
- DirectInput exposes X/Y/Z, Rx/Ry/Rz, Slider 1/2 plus buttons/POVs.
- Pedal mode Pitch maps the normalized pedal axis to virtual Xbox Left Stick Y.
- Learned PITCHUP / PITCHDOWN bindings generate full-deflection Left Stick Y outputs.
- Digital pitch bindings override analog Left Stick Y while held; both together cancel to neutral.
- Pedal Keyboard Keys mode supports modifier keys including Shift, left/right Control and Alt.

The Win32 UI remains a fixed-size content canvas inside a resizable/scrollable main window.
- Twist Lever Output Negative multiplies the deadzone-corrected output by -1;
  Invert still reverses the one-way travel before deadzone. Settings persist
  as twist_left.output_negative and twist_right.output_negative (default false).
- axis_math_test.go checks the invert/polarity combinations and older JSON.


v2.1.0-beta UI Notes
--------------------
- Axis tuning options are contextual to Podracer, Traditional HOTAS and Driving Wheel control modes.
- Podracer Left/Right inputs show deadzone only; output curves are shared
  for combined thrust/steering so individual stick curves cannot skew turns.
- Axis tuning now lives in the right-side tab; slider edits auto-apply and are saved after about 450 ms of inactivity (or when switching away/exiting). Reset This Axis also applies immediately; Apply/Close no longer exist.
- New installs and first-time v2 migrations start with Hybrid Preserve Thrust on.
- Main Driving section now follows Pedals and precedes On-foot mappings.


v2.0.8 BETA DEVELOPMENT NOTES
-----------------------------
- New driving_wheel JSON subtree stores Wheel, Pedal 1 (RT), Pedal 2 (LT),
  Pedal 3 (optional Off/Boost/Drift) and Hand Brake assignments independently.
- Wheel and all pedal axes have separate tuning keys in axis_tuning_math.go.
- Hand Brake mode is Boost or Drift; optional digital button binding key is
  WHEELHANDBRAKE. Old user bindings remain untouched.
- Centered Boost/Drift and Lever Boost/Drift output full virtual Xbox A/X
  after the configured axis deadzone has been applied.
- wheel_math.go has platform-neutral output helpers and portable tests.
- New driving_wheel_windows_test.go is cross-compiled as part of Windows test
  build, but is not run unless on a Windows system.
- Changing to Driving Wheel shifts the Additional/Hand Brake blocks 45px;
  switching back restores the original v2.0.6 geometry in HOSAS/HOTAS.
- Test the source on real hardware and Windows 100%/125%/150% scale.


v2.1.0-beta TAB / MONITOR NOTES
--------------------------------
- Right-side Mappings and Input Curves/Axis Tuning use native Windows tabs.
  Mappings opens first; tab state survives mode switching within a session.
- The tuning page is a Win32 child window, NOT a separate top-level window.
  Slider changes directly update app.cfg.Tuning while pending file writes coalesce.
- Calibration polling rate and Podracer-only Hybrid Preserve Thrust are now
  mounted on the main window and hidden with the tuning tab.
- HOTAS: Throttle/Brake plot then Stick plot. Driving Wheel: Throttle/Brake
  with positive throttle / negative brake, then horizontally centered Wheel.
- HOTAS device diagnostic is physically below all three plot/caption areas.
- Installer does not alter hosas_config.json; game controller scheme should be Default.
- Re-test Win32 parent/child rendering, settings switching and DPI on Windows.


v2.1.4 calibration update: In Two Stage mode, normal Midpoint Position is now
a 10–90% slider with a live percentage readout. Enabling Custom Midpoint
hides the diagonal slider and (in v2.1.4) showed midpoint presets
alongside independent X Input / Y Output sliders. Presets are removed in v2.1.5. Calibration saves automatically;
curve calculations and saved configuration keys are unchanged.
