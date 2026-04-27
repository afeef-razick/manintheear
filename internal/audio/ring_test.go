package audio

import (
	"testing"
)

func TestRingBuffer_OrderedBeforeFull(t *testing.T) {
	rb := newRingBuffer()
	samples := make([]float32, 100)
	for i := range samples {
		samples[i] = float32(i)
	}
	rb.write(samples)

	out := rb.drain()
	if len(out) != 100 {
		t.Fatalf("drain len = %d, want 100", len(out))
	}
	for i, v := range out {
		if v != float32(i) {
			t.Fatalf("out[%d] = %v, want %v", i, v, float32(i))
		}
	}
}

func TestRingBuffer_WrapsCorrectly(t *testing.T) {
	rb := newRingBuffer()

	// fill past the window boundary
	first := make([]float32, windowSamples)
	for i := range first {
		first[i] = 1.0
	}
	rb.write(first)

	second := make([]float32, 100)
	for i := range second {
		second[i] = 2.0
	}
	rb.write(second)

	out := rb.drain()
	if len(out) != windowSamples {
		t.Fatalf("drain len = %d, want %d", len(out), windowSamples)
	}

	// last 100 samples should be 2.0, rest should be 1.0
	for i, v := range out[windowSamples-100:] {
		if v != 2.0 {
			t.Errorf("tail[%d] = %v, want 2.0", i, v)
		}
	}
	for i, v := range out[:windowSamples-100] {
		if v != 1.0 {
			t.Errorf("body[%d] = %v, want 1.0", i, v)
		}
	}
}
