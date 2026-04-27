package audio

import (
	"context"
	"fmt"

	"github.com/gordonklaus/portaudio"
)

const framesPerBuffer = 512

type Capture struct {
	stream *portaudio.Stream
	ring   *ringBuffer
}

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

func (c *Capture) Drain() []byte {
	return EncodeWAV(c.ring.drain(), SampleRate)
}

func (c *Capture) Close() error {
	if err := c.stream.Close(); err != nil {
		return fmt.Errorf("audio: close stream: %w", err)
	}
	if err := portaudio.Terminate(); err != nil {
		return fmt.Errorf("audio: portaudio terminate: %w", err)
	}
	return nil
}
