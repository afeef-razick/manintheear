package audio_test

import (
	"encoding/binary"
	"testing"

	"github.com/afeef-razick/manintheear/internal/audio"
)

func TestEncodeWAV_Header(t *testing.T) {
	samples := make([]float32, 100)
	wav := audio.EncodeWAV(samples, 16000)

	if string(wav[0:4]) != "RIFF" {
		t.Errorf("RIFF marker = %q, want %q", wav[0:4], "RIFF")
	}
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("WAVE marker = %q, want %q", wav[8:12], "WAVE")
	}
	if string(wav[12:16]) != "fmt " {
		t.Errorf("fmt marker = %q, want %q", wav[12:16], "fmt ")
	}
	if string(wav[36:40]) != "data" {
		t.Errorf("data marker = %q, want %q", wav[36:40], "data")
	}
}

func TestEncodeWAV_SampleRate(t *testing.T) {
	samples := make([]float32, 100)
	wav := audio.EncodeWAV(samples, 16000)

	// sample rate is at byte 24
	rate := binary.LittleEndian.Uint32(wav[24:28])
	if rate != 16000 {
		t.Errorf("sample rate = %d, want 16000", rate)
	}
}

func TestEncodeWAV_DataSize(t *testing.T) {
	n := 200
	samples := make([]float32, n)
	wav := audio.EncodeWAV(samples, 16000)

	// data chunk size at byte 40: n samples * 2 bytes each
	dataSize := binary.LittleEndian.Uint32(wav[40:44])
	if int(dataSize) != n*2 {
		t.Errorf("data size = %d, want %d", dataSize, n*2)
	}
}

func TestEncodeWAV_Clamp(t *testing.T) {
	samples := []float32{2.0, -2.0, 0.0}
	wav := audio.EncodeWAV(samples, 16000)

	// first sample should clamp to max int16
	v0 := int16(binary.LittleEndian.Uint16(wav[44:46]))
	if v0 != 32767 {
		t.Errorf("over-range sample = %d, want 32767", v0)
	}

	// second sample should clamp to min int16
	v1 := int16(binary.LittleEndian.Uint16(wav[46:48]))
	if v1 != -32767 {
		t.Errorf("under-range sample = %d, want -32767", v1)
	}

	// silence
	v2 := int16(binary.LittleEndian.Uint16(wav[48:50]))
	if v2 != 0 {
		t.Errorf("zero sample = %d, want 0", v2)
	}
}
