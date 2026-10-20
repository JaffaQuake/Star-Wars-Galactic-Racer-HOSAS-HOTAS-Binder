package main

import (
	"math"
	"strings"
)

// Axis response is adjusted after physical normalization and deadzone removal.
// Curve=1 and FullOutput=100 preserve linear full-range behavior.
type axisTuneProfile struct {
	Deadzone       int     `json:"deadzone"`
	Curve          float64 `json:"curve"`               // single-curve response strength, legacy compatible
	FullOutput     int     `json:"full_output"`         // shared full-output point for either mode
	Weighting      *int    `json:"weighting,omitempty"` // nil in old configs means legacy power curve (0%)
	TwoStage       bool    `json:"two_stage,omitempty"`
	Stage1Curve    float64 `json:"stage1_curve,omitempty"`
	Stage2Curve    float64 `json:"stage2_curve,omitempty"`
	Stage1Weight   *int    `json:"stage1_weight,omitempty"`
	Stage2Weight   *int    `json:"stage2_weight,omitempty"`
	Midpoint       int     `json:"midpoint,omitempty"` // ordinary diagonal %: X = Y
	CustomMidpoint bool    `json:"custom_midpoint,omitempty"`
	MidpointX      int     `json:"midpoint_x,omitempty"`
	MidpointY      int     `json:"midpoint_y,omitempty"`
}

func axisInt(v int) *int { return &v }
func axisWeight(v *int) float64 {
	if v == nil {
		return 0
	} // preserve older users' power/exponent calibration
	return math.Max(0, math.Min(100, float64(*v))) / 100
}
func displayAxisWeight(v *int) int {
	if v == nil {
		return 0
	}
	return int(math.Max(0, math.Min(100, float64(*v))))
}

const (
	profileHOSASLeft      = "hosas_left"
	profileHOSASRight     = "hosas_right"
	profileHOSASThrust    = "hosas_thrust"
	profileHOSASSteer     = "hosas_steering"
	profileHOTASX         = "hotas_x"
	profileHOTASY         = "hotas_y"
	profileHOTASThrottle  = "hotas_throttle"
	profilePedal          = "pedals"
	profileTwistLeft      = "twist_left"
	profileTwistRight     = "twist_right"
	profileWheelSteering  = "wheel_steering"
	profileWheelPedal1    = "wheel_pedal_1"
	profileWheelPedal2    = "wheel_pedal_2"
	profileWheelPedal3    = "wheel_pedal_3"
	profileWheelHandbrake = "wheel_handbrake"
)

type axisTuneEntry struct {
	Key, Name    string
	DeadzoneOnly bool
	OneWay       bool
}

// All saved profiles must remain available for migration even when not shown
// in the popup for the current control mode.
var axisTuneNames = []axisTuneEntry{
	{profileHOSASLeft, "Left Input", true, false},
	{profileHOSASRight, "Right Input", true, false},
	{profileHOSASThrust, "Thrust Output", false, false},
	{profileHOSASSteer, "Steering Output", false, false},
	{profileHOTASX, "Stick X", false, false},
	{profileHOTASY, "Stick Y", false, false},
	{profileHOTASThrottle, "Throttle", false, true},
	{profilePedal, "Rudder Pedals", false, false},
	{profileTwistLeft, "Additional Input 1", false, false},
	{profileTwistRight, "Additional Input 2", false, false},
	{profileWheelSteering, "Wheel Steering", false, false},
	{profileWheelPedal1, "Pedal 1 - Throttle", false, true},
	{profileWheelPedal2, "Pedal 2 - Brake", false, true},
	{profileWheelPedal3, "Pedal 3 - Optional", false, true},
	{profileWheelHandbrake, "Hand Brake", false, true},
}

// Common auxiliary controls are displayed in both modes; only the mode's
// own stick and output profiles appear in its Input to Configure selector.
func axisTuneEntriesForControlMode(mode string) []axisTuneEntry {
	entries := make([]axisTuneEntry, 0, 10)
	for _, entry := range axisTuneNames {
		hosas := entry.Key == profileHOSASLeft || entry.Key == profileHOSASRight || entry.Key == profileHOSASThrust || entry.Key == profileHOSASSteer
		hotas := entry.Key == profileHOTASX || entry.Key == profileHOTASY || entry.Key == profileHOTASThrottle
		wheel := entry.Key == profileWheelSteering || entry.Key == profileWheelPedal1 || entry.Key == profileWheelPedal2 || entry.Key == profileWheelPedal3 || entry.Key == profileWheelHandbrake
		if hosas && mode != "HOSAS" && mode != "HOTAT" || hotas && mode != "HOTAS" && mode != "SINGLE" || wheel && mode != "WHEEL" {
			continue
		}
		if entry.Key == profileHOTASThrottle && mode == "SINGLE" {
			continue
		}
		if entry.Key == profilePedal && mode == "WHEEL" {
			continue
		} // Hand Brake uses its own lever tune
		entries = append(entries, entry)
	}
	return entries
}

// Compatibility wrapper for existing callers/tests.
func axisTuneEntriesForMode(isHOTAS bool) []axisTuneEntry {
	if isHOTAS {
		return axisTuneEntriesForControlMode("HOTAS")
	}
	return axisTuneEntriesForControlMode("HOSAS")
}

func defaultAxisTune(dz int) axisTuneProfile {
	return axisTuneProfile{Deadzone: dz, Curve: 1, FullOutput: 100,
		Weighting: axisInt(50), Stage1Curve: 1, Stage2Curve: 1,
		Stage1Weight: axisInt(50), Stage2Weight: axisInt(50),
		Midpoint: 50, MidpointX: 50, MidpointY: 50}
}

func checkedAxisTune(p axisTuneProfile) axisTuneProfile {
	if p.Deadzone < 0 || p.Deadzone > 40 {
		p.Deadzone = 6
	}
	if math.IsNaN(p.Curve) || math.IsInf(p.Curve, 0) || p.Curve < 0.2 || p.Curve > 3 {
		p.Curve = 1
	}
	if p.FullOutput < 50 || p.FullOutput > 100 {
		p.FullOutput = 100
	}
	// Legacy non-linear calibrations retain their exact power/exponent output.
	// Old linear profiles may safely initialize the new visible weight at center.
	if p.Weighting == nil && p.Curve == 1 {
		p.Weighting = axisInt(50)
	}
	if p.Stage1Curve < 0.2 || p.Stage1Curve > 3 || math.IsNaN(p.Stage1Curve) || math.IsInf(p.Stage1Curve, 0) {
		p.Stage1Curve = 1
	}
	if p.Stage2Curve < 0.2 || p.Stage2Curve > 3 || math.IsNaN(p.Stage2Curve) || math.IsInf(p.Stage2Curve, 0) {
		p.Stage2Curve = 1
	}
	if p.Stage1Weight == nil {
		p.Stage1Weight = axisInt(50)
	}
	if p.Stage2Weight == nil {
		p.Stage2Weight = axisInt(50)
	}
	if p.Midpoint < 10 || p.Midpoint > 90 {
		p.Midpoint = 50
	}
	if p.MidpointX < 10 || p.MidpointX > 90 {
		p.MidpointX = 50
	}
	if p.MidpointY < 10 || p.MidpointY > 90 {
		p.MidpointY = 50
	}
	return p
}

// Smooth weighted power arc as approved in the final interactive simulation.
// Weighting 0 is precisely the original power curve (existing configs).
// Weighting 50 blends two smooth same-concavity arcs; no internal corners.
func weightedArc(x, power, weight float64) float64 {
	x = math.Max(0, math.Min(1, x))
	if x == 0 || x == 1 || power == 1 {
		return x
	}
	weight = math.Max(0, math.Min(1, weight))
	return (1-weight)*math.Pow(x, power) + weight*(1-math.Pow(1-x, 1/power))
}
func weightedArcDerivative(x, power, weight float64) float64 {
	weight = math.Max(0, math.Min(1, weight))
	return (1-weight)*power*math.Pow(x, power-1) + weight/power*math.Pow(1-x, 1/power-1)
}
func curveHermite(x, x0, y0, x1, y1, d0, d1 float64) float64 {
	u := (x - x0) / (x1 - x0)
	u2, u3 := u*u, u*u*u
	return (2*u3-3*u2+1)*y0 + (u3-2*u2+u)*(x1-x0)*d0 + (-2*u3+3*u2)*y1 + (u3-u2)*(x1-x0)*d1
}

// Two stage arc with C1-continuity at the midpoint. Both Hermite pieces use
// slope-limited endpoints to stay monotonic even for extreme custom midpoints
// and curve settings (the original simulation did not guard these extremes).
func twoStageArc(x float64, p axisTuneProfile) float64 {
	mx, my := float64(p.Midpoint), float64(p.Midpoint)
	if p.CustomMidpoint {
		mx, my = float64(p.MidpointX), float64(p.MidpointY)
	}
	x *= 100 // 0..100 for legible midpoint arithmetic
	left := func(t float64) float64 { return my * weightedArc(t/mx, p.Stage1Curve, axisWeight(p.Stage1Weight)) }
	right := func(t float64) float64 {
		return my + (100-my)*weightedArc((t-mx)/(100-mx), p.Stage2Curve, axisWeight(p.Stage2Weight))
	}
	h := math.Min(9, math.Min(mx*0.3, (100-mx)*0.3))
	a, b := mx-h, mx+h
	if x <= a {
		return left(x) / 100
	}
	if x >= b {
		return right(x) / 100
	}
	ya, yb := left(a), right(b)
	sl, sr := (my-ya)/h, (yb-my)/h
	dm := 2 * sl * sr / (sl + sr)
	dl := my / mx * weightedArcDerivative(a/mx, p.Stage1Curve, axisWeight(p.Stage1Weight))
	dr := (100 - my) / (100 - mx) * weightedArcDerivative((b-mx)/(100-mx), p.Stage2Curve, axisWeight(p.Stage2Weight))
	// Sufficient monotone cubic Hermite constraint: normalized derivatives sum <= 3.
	dl = math.Max(0, math.Min(dl, 3*sl-dm))
	dr = math.Max(0, math.Min(dr, 3*sr-dm))
	if x < mx {
		return curveHermite(x, a, ya, mx, my, dl, dm) / 100
	}
	return curveHermite(x, mx, my, b, yb, dm, dr) / 100
}

func tunedValueAfterDeadzone(v float64, p axisTuneProfile) float64 {
	p = checkedAxisTune(p)
	mag := math.Min(1, math.Abs(v)/(float64(p.FullOutput)/100))
	result := weightedArc(mag, p.Curve, axisWeight(p.Weighting))
	if p.TwoStage {
		result = twoStageArc(mag, p)
	}
	return math.Copysign(math.Max(0, math.Min(1, result)), v)
}

// One-way input 0..1: deadzone trims near zero and rescales travel, then curves it.
func tunedOneWay(v float64, p axisTuneProfile) float64 {
	p = checkedAxisTune(p)
	v = math.Max(0, math.Min(1, v))
	dz := float64(p.Deadzone) / 100
	if v <= dz {
		return 0
	}
	v = (v - dz) / (1 - dz)
	return tunedValueAfterDeadzone(v, p)
}

// Centered input -1..+1: same response on each side of center.
func tunedCentered(v float64, p axisTuneProfile) float64 {
	p = checkedAxisTune(p)
	dz := float64(p.Deadzone) / 100
	mag := math.Min(1, math.Abs(v))
	if mag <= dz {
		return 0
	}
	mag = (mag - dz) / (1 - dz)
	return tunedValueAfterDeadzone(math.Copysign(mag, v), p)
}

// Clone before mutating profiles so the polling goroutine can safely read
// the previous immutable map snapshot while settings are being applied.
func cloneAxisProfiles(src map[string]axisTuneProfile) map[string]axisTuneProfile {
	out := make(map[string]axisTuneProfile, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// Camera diagnostics are available only while at least one auxiliary input
// (Rudder Pedals or either Twist Axis) is assigned to Move Camera.
func cameraTestButtonsVisible(modes ...string) bool {
	for _, mode := range modes {
		if strings.EqualFold(mode, "Move Camera") {
			return true
		}
	}
	return false
}

// Camera test controls require both an active controller and a matching
// auxiliary Move Camera mode. Keep the mode logic portable for unit tests.
func cameraTestControlsVisible(controllerActive bool, modes ...string) bool {
	return controllerActive && cameraTestButtonsVisible(modes...)
}

// The selector should only offer active, assigned axes with relevant controls.
// Keep dormant profile values in the config map for later reactivation.
type tuneInputInfo struct {
	Enabled     bool
	DigitalOnly bool
	OneWay      bool
}

func isTuningDigitalMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "boost", "drift", "boost/drift", "keyboard keys":
		return true
	}
	return false
}
func filterAxisTuneEntries(entries []axisTuneEntry, statuses map[string]tuneInputInfo) []axisTuneEntry {
	out := make([]axisTuneEntry, 0, len(entries))
	for _, e := range entries {
		status := statuses[e.Key]
		if !status.Enabled {
			continue
		}
		if status.DigitalOnly {
			e.DeadzoneOnly = true
		}
		if status.OneWay {
			e.OneWay = true
		}
		out = append(out, e)
	}
	return out
}
