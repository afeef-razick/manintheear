package audio

import "sync"

const (
	SampleRate    = 16000
	windowSeconds = 10
	windowSamples = SampleRate * windowSeconds
)

type ringBuffer struct {
	data []float32
	head int
	full bool
	mu   sync.Mutex
}

func newRingBuffer() *ringBuffer {
	return &ringBuffer{data: make([]float32, windowSamples)}
}

func (r *ringBuffer) write(samples []float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range samples {
		r.data[r.head] = s
		r.head = (r.head + 1) % windowSamples
		if r.head == 0 {
			r.full = true
		}
	}
}

func (r *ringBuffer) drain() []float32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]float32, windowSamples)
	if !r.full {
		copy(out, r.data[:r.head])
		return out[:r.head]
	}
	n := copy(out, r.data[r.head:])
	copy(out[n:], r.data[:r.head])
	return out
}
