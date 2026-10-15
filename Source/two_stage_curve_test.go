package main

import (
	"encoding/json"
	"math"
	"math/rand"
	"testing"
)

func TestSmoothWeightedArcEndpointsAndMonotonicity(t *testing.T) {
	for _, p := range []float64{.2, .35, .5, .85, 1, 1.25, 2, 3} {
		for w := 0; w <= 100; w += 5 {
			weight := float64(w) / 100
			near(t, weightedArc(0, p, weight), 0)
			near(t, weightedArc(1, p, weight), 1)
			last := -1.0
			for i := 0; i <= 1000; i++ {
				x := float64(i) / 1000
				y := weightedArc(x, p, weight)
				if y < last-1e-10 || y < -1e-10 || y > 1+1e-10 {
					t.Fatalf("bad smooth arc: p=%v w=%d x=%v y=%v previous=%v", p, w, x, y, last)
				}
				last = y
			}
			if p == 1 {
				near(t, weightedArc(.3, p, weight), .3)
			}
		}
	}
}
func TestLegacySingleCurveIsPreservedByMigration(t *testing.T) {
	var p axisTuneProfile
	if err := json.Unmarshal([]byte(`{"deadzone":0,"curve":2,"full_output":80}`), &p); err != nil {
		t.Fatal(err)
	}
	p = checkedAxisTune(p)
	if p.Weighting != nil || p.TwoStage {
		t.Fatal("legacy settings must retain zero weighting and one stage")
	}
	near(t, tunedCentered(.4, p), .25)
	near(t, tunedCentered(.8, p), 1)
	newProfile := defaultAxisTune(0)
	if displayAxisWeight(newProfile.Weighting) != 50 || newProfile.TwoStage {
		t.Fatal("new profile defaults must be centered single-stage")
	}
}
func TestTwoStageCurveMidpointsAndGlobalFullOutput(t *testing.T) {
	p := defaultAxisTune(0)
	p.TwoStage = true
	p.Stage1Curve = .2
	p.Stage2Curve = 3
	p.Stage1Weight = axisInt(25)
	p.Stage2Weight = axisInt(80)
	p.Midpoint = 60
	p.FullOutput = 80
	near(t, tunedOneWay(.8*.6, p), .6)
	near(t, tunedOneWay(.8, p), 1)
	near(t, tunedOneWay(1, p), 1)
	p.CustomMidpoint = true
	p.MidpointX = 35
	p.MidpointY = 65
	near(t, tunedOneWay(.8*.35, p), .65)
}
func TestTwoStageRandomMonotonicAndC1Join(t *testing.T) {
	rng := rand.New(rand.NewSource(130709))
	for n := 0; n < 1600; n++ {
		p := defaultAxisTune(0)
		p.TwoStage = true
		p.CustomMidpoint = true
		p.MidpointX = 10 + rng.Intn(81)
		p.MidpointY = 10 + rng.Intn(81)
		p.Stage1Curve = .2 + 2.8*rng.Float64()
		p.Stage2Curve = .2 + 2.8*rng.Float64()
		p.Stage1Weight = axisInt(rng.Intn(101))
		p.Stage2Weight = axisInt(rng.Intn(101))
		last := -1.0
		for j := 0; j <= 300; j++ {
			x := float64(j) / 300
			y := twoStageArc(x, p)
			if math.IsNaN(y) || math.IsInf(y, 0) || y < last-1e-8 || y > 1+1e-8 {
				t.Fatalf("nonmonotonic curve profile=%+v x=%v y=%v last=%v", p, x, y, last)
			}
			last = y
		}
		at := float64(p.MidpointX) / 100
		near(t, twoStageArc(at, p), float64(p.MidpointY)/100)
		h := 0.000001
		dLeft := (twoStageArc(at, p) - twoStageArc(at-h, p)) / h
		dRight := (twoStageArc(at+h, p) - twoStageArc(at, p)) / h
		if math.Abs(dLeft-dRight) > 0.035 {
			t.Fatalf("midpoint has sharp corner: %v v %v profile=%+v", dLeft, dRight, p)
		}
	}
}
func TestNewCurveSettingsJSONRoundTrip(t *testing.T) {
	p := defaultAxisTune(6)
	p.TwoStage = true
	p.Weighting = axisInt(0)
	p.Stage1Curve = .2
	p.Stage2Curve = 2.75
	p.Stage1Weight = axisInt(13)
	p.Stage2Weight = axisInt(87)
	p.CustomMidpoint = true
	p.MidpointX = 37
	p.MidpointY = 66
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var back axisTuneProfile
	if err = json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	back = checkedAxisTune(back)
	if !back.TwoStage || !back.CustomMidpoint || back.MidpointX != 37 || back.MidpointY != 66 || displayAxisWeight(back.Weighting) != 0 || displayAxisWeight(back.Stage1Weight) != 13 || displayAxisWeight(back.Stage2Weight) != 87 || back.Stage1Curve != .2 || back.Stage2Curve != 2.75 {
		t.Fatalf("settings did not survive: %+v", back)
	}
}
