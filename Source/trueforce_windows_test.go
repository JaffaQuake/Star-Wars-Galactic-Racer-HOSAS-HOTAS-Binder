package main

import (
	"encoding/hex"
	"math"
	"testing"
)

func TestTFPacketLayout(t *testing.T) {
	var pkt [64]byte
	var w [tfWindowSlots]uint16
	// Protocol doc example: neutral, sequence 0x7F -> 01 00 00 00 01 7F 00 80 00 80 then zeros.
	buildTFPacket(&pkt, 0x7F, tfU16(0), &w, 0)
	if got := hex.EncodeToString(pkt[:12]); got != "01000000017f008000800000" {
		t.Fatalf("idle packet = %s", got)
	}
	for _, b := range pkt[10:] {
		if b != 0 {
			t.Fatal("idle packet must have byte10/11 and tail zero")
		}
	}
	if tfU16(-1) != 0x0001 || tfU16(1) != 0xFFFF || tfU16(-5) != 0x0001 {
		t.Fatalf("tfU16 extremes: %04x %04x", tfU16(-1), tfU16(1))
	}
	for i := range w {
		w[i] = uint16(0x8000 + i)
	}
	buildTFPacket(&pkt, 3, 0x1234, &w, 4)
	if pkt[10] != 4 || pkt[11] != 0x0d {
		t.Fatalf("sample packet demux pair = %02x %02x", pkt[10], pkt[11])
	}
	if pkt[6] != 0x34 || pkt[7] != 0x12 || pkt[8] != 0x34 || pkt[9] != 0x12 {
		t.Fatal("cur must be LE and duplicated")
	}
	if pkt[60] != 0x0c || pkt[61] != 0x80 || pkt[62] != 0x0c || pkt[63] != 0x80 {
		t.Fatalf("newest sample not in slot 12 L/R: % x", pkt[60:])
	}
}

func TestTFSpringPushesTowardCentre(t *testing.T) {
	// Wheel turned left (negative) must get a rightward (positive) force, and vice versa.
	if f := tfSpring(-0.2, 0, 45); f <= 0 || math.Abs(f-0.09) > 1e-9 {
		t.Fatalf("left of centre: %v", f)
	}
	if f := tfSpring(0.2, 0, 45); f >= 0 {
		t.Fatalf("right of centre: %v", f)
	}
	if f := tfSpring(0, 1, 45); f >= 0 {
		t.Fatalf("damping must oppose motion: %v", f)
	}
}

func TestTFRunawayOnlyWhenPushedOutAtLock(t *testing.T) {
	// Driver holding full lock while the spring pulls back is normal driving.
	if tfRunaway(-0.99, tfSpring(-0.99, 0, 35)) || tfRunaway(0.99, tfSpring(0.99, 0, 35)) {
		t.Fatal("held at lock against the spring must not trip")
	}
	if !tfRunaway(-0.99, -0.3) || !tfRunaway(0.99, 0.3) {
		t.Fatal("force pushing further into lock must trip")
	}
	if tfRunaway(-0.5, -0.3) {
		t.Fatal("away from lock must not trip")
	}
}

func TestTFSpringDampingBounded(t *testing.T) {
	// A poll-timing glitch can report absurd speed; damping must not kick the rim.
	if f := tfSpring(0, 500, 35); math.Abs(f) > 0.1 {
		t.Fatalf("damping kick %.2f", f)
	}
}

func TestTFDecimatorBand(t *testing.T) {
	rms := func(hz float64) float64 {
		d := newTFDecimator(48000)
		s := make([]float32, 0, 2*48000/5)
		for i := 0; i < 48000/5; i++ {
			v := float32(0.5 * math.Sin(2*math.Pi*hz*float64(i)/48000))
			s = append(s, v, v)
		}
		var sum float64
		n := 0
		d.feed(s, 2, func(v float64) {
			if n++; n > 100 { // skip filter settling
				sum += v * v
			}
		})
		if n != 800 {
			t.Fatalf("expected 800 samples at 4 kHz from 200 ms, got %d", n)
		}
		return math.Sqrt(sum / float64(n-100))
	}
	if pass, dc, high := rms(200), rms(2), rms(10000); pass < 0.25 || dc > 0.05 || high > 0.05 {
		t.Fatalf("band-pass off: 200Hz=%.3f 2Hz=%.3f 10kHz=%.3f", pass, dc, high)
	}
}

func TestTFRingJitterBuffer(t *testing.T) {
	r := &tfRing{buffering: true}
	for i := 0; i < tfRingStart-1; i++ {
		r.push(1)
	}
	if got := r.pop(4); got != nil {
		t.Fatal("must hold back until the jitter buffer fills")
	}
	r.push(1)
	if got := r.pop(4); len(got) != 4 {
		t.Fatalf("got %d samples once filled", len(got))
	}
	for i := 0; i < tfRingMax*2; i++ {
		r.push(float64(i))
	}
	if len(r.buf) != tfRingMax || r.buf[len(r.buf)-1] != float64(tfRingMax*2-1) {
		t.Fatal("overflow must drop the oldest samples")
	}
}

func TestTFInitSequence(t *testing.T) {
	if len(tfInitPackets) != 68 {
		t.Fatalf("init has %d packets, want 68", len(tfInitPackets))
	}
	types := map[byte]int{}
	for _, hx := range tfInitPackets {
		b, err := hex.DecodeString(hx)
		if err != nil || len(b) < 5 || len(b) > 64 || b[0] != 0x01 {
			t.Fatalf("bad init packet %q", hx)
		}
		types[b[4]]++
	}
	if types[0x05] != 48 || types[0x06] != 6 || types[0x01] != 9 {
		t.Fatalf("init type counts %v", types)
	}
	last := tfInitPackets[66:]
	if last[0] != "0100000004" || last[1] != "0100000003" {
		t.Fatal("init must end with stop then start")
	}
}
