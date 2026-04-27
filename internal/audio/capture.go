package audio

import (
	"context"
	"fmt"

	"github.com/gordonklaus/portaudio"
)

const framesPerBuffer = 512

// Capture records mono audio from the default input device into a ring buffer.
type Capture struct {
	stream *portaudio.Stream
	ring   *ringBuffer
}

// New initialises portaudio and opens the default input stream.
// Call Close when done to release audio hardware.
func New() (*Capture, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("audio: portaudio init: %w", err)
	}

	c := &Capture{ring: newRingBuffer()}

	stream, err := portaudio.OpenDefaultStream(1, 0, float64(SampleRate), framesPerBuffer, func(in []float32) {
		c.ring.write(in)
	})
	if err != nil {
		portaudio.Terminate() //nolint:errcheck
		return nil, fmt.Errorf("audio: open stream: %w", err)
	}

	c.stream = stream
	return c, nil
}

// Start begins recording. It blocks until ctx is cancelled, then stops the stream.
func (c *Capture) Start(ctx context.Context) error {
	if err := c.stream.Start(); err != nil {
		return fmt.Errorf("audio: start stream: %w", err)
	}
	<-ctx.Done()
	if err := c.stream.Stop(); err != nil {
		return fmt.Errorf("audio: stop stream: %w", err)
	}
	return nil
}

// Drain returns WAV-encoded audio for the last window of captured samples.
func (c *Capture) Drain() []byte {
	return EncodeWAV(c.ring.drain(), SampleRate)
}

// Close releases the portaudio stream and terminates the audio subsystem.
func (c *Capture) Close() error {
	if err := c.stream.Close(); err != nil {
		return fmt.Errorf("audio: close stream: %w", err)
	}
	if err := portaudio.Terminate(); err != nil {
		return fmt.Errorf("audio: portaudio terminate: %w", err)
	}
	return nil
}
