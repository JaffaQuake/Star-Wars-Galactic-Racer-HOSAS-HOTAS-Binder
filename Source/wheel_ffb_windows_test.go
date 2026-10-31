package main

import (
	"math"
	"testing"
	"time"
)

func TestAudioMenuGainFadesOffThrottle(t *testing.T) {
	now := time.Now()
	lastThrottleAt.Store(0)
	if g := audioMenuGain(now); g != 0 {
		t.Fatalf("never on throttle (menus at launch) = %v", g)
	}
	noteThrottle(10)
	if g := audioMenuGain(now); g != 0 {
		t.Fatalf("light trigger touch must not count as driving: %v", g)
	}
	noteThrottle(200)
	at := time.Unix(0, lastThrottleAt.Load())
	for _, c := range []struct {
		after time.Duration
		want  float64
	}{{0, 1}, {menuIdleAfter, 1}, {menuIdleAfter + menuFade/2, 0.5}, {menuIdleAfter + menuFade, 0}} {
		if g := audioMenuGain(at.Add(c.after)); math.Abs(g-c.want) > 1e-6 {
			t.Fatalf("%v off throttle: gain %v, want %v", c.after, g, c.want)
		}
	}
}

func TestWheelFFBScaling(t *testing.T) {
	for _, c := range []struct{ v, want int }{{0, 50}, {-1, 0}, {150, 100}, {25, 25}} {
		if got := ffbPercent(c.v, 50); got != c.want {
			t.Errorf("ffbPercent(%d) = %d, want %d", c.v, got, c.want)
		}
	}
	if got := wheelFFBMagnitudes(255, 0, 255, 100, 50, 30); got != [4]int32{3000, 10000, 0, 5000} {
		t.Errorf("full motor = %v", got)
	}
	if got := wheelFFBMagnitudes(255, 255, 255, 0, 0, 0); got != [4]int32{} {
		t.Errorf("off = %v", got)
	}
}

func TestAudioEnvelopeFollowsBass(t *testing.T) {
	tone := func(hz float64) []float32 { // 100 ms stereo at 48 kHz, amplitude 0.3
		s := make([]float32, 0, 9600)
		for i := 0; i < 4800; i++ {
			v := float32(0.3 * math.Sin(2*math.Pi*hz*float64(i)/48000))
			s = append(s, v, v)
		}
		return s
	}
	bass := newAudioEnvelope(48000)
	bass.feed(tone(60), 2)
	treble := newAudioEnvelope(48000)
	treble.feed(tone(4000), 2)
	if bass.byteLevel() < 150 || treble.byteLevel() > 20 {
		t.Fatalf("bass=%d treble=%d: low-pass not isolating rumble band", bass.byteLevel(), treble.byteLevel())
	}
	for i := 0; i < 50; i++ {
		bass.push(0)
	}
	if bass.byteLevel() != 0 {
		t.Fatalf("level did not decay to 0 in silence: %d", bass.byteLevel())
	}
}
