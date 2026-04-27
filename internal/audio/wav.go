package audio

import "encoding/binary"

func EncodeWAV(samples []float32, sampleRate int) []byte {
	const (
		channels      = 1
		bitsPerSample = 16
		headerSize    = 44
	)
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	blockAlign := uint16(channels * bitsPerSample / 8)
	dataSize := uint32(len(samples) * 2)
	riffSize := 36 + dataSize

	out := make([]byte, headerSize+int(dataSize))

	copy(out[0:], "RIFF")
	binary.LittleEndian.PutUint32(out[4:], riffSize)
	copy(out[8:], "WAVE")
	copy(out[12:], "fmt ")
	binary.LittleEndian.PutUint32(out[16:], 16)
	binary.LittleEndian.PutUint16(out[20:], 1) // PCM
	binary.LittleEndian.PutUint16(out[22:], channels)
	binary.LittleEndian.PutUint32(out[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(out[28:], byteRate)
	binary.LittleEndian.PutUint16(out[32:], blockAlign)
	binary.LittleEndian.PutUint16(out[34:], bitsPerSample)
	copy(out[36:], "data")
	binary.LittleEndian.PutUint32(out[40:], dataSize)

	for i, s := range samples {
		if s > 1.0 {
			s = 1.0
		} else if s < -1.0 {
			s = -1.0
		}
		binary.LittleEndian.PutUint16(out[headerSize+i*2:], uint16(int16(s*32767)))
	}

	return out
}
