package sipnexus

import (
	"fmt"
	"sync"

	audiomultiplexer "github.com/itzmanish/audio-multiplexer-go"
	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/pion/rtp"
)

type TranscodingService struct {
	logger logger.Logger
	mixers map[string]*audiomultiplexer.Mixer
	mu     sync.RWMutex
}

func NewTranscodingService(logger logger.Logger) *TranscodingService {
	return &TranscodingService{
		logger: logger,
		mixers: make(map[string]*audiomultiplexer.Mixer),
	}
}

func (t *TranscodingService) TranscodePacket(packet *rtp.Packet, sourceCodec, targetCodec string) (*rtp.Packet, error) {
	t.logger.Info(fmt.Sprintf("Transcoding from %s to %s", sourceCodec, targetCodec))

	// Get or create a mixer for this transcoding operation
	mixerKey := fmt.Sprintf("%s-%s", sourceCodec, targetCodec)
	mixer := t.getMixer(mixerKey)

	// Convert RTP packet to samples
	samples, err := t.rtpToSamples(packet, sourceCodec)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RTP to samples: %w", err)
	}

	// Add samples to the mixer
	mixer.AddSamples("source", samples)

	// Mix (in this case, it's just passing through as we're not actually mixing)
	mixedSamples := mixer.Mix()

	// Convert mixed samples back to RTP packet with the target codec
	transcodedPacket, err := t.samplesToRTP(mixedSamples, packet.Header, targetCodec)
	if err != nil {
		return nil, fmt.Errorf("failed to convert samples to RTP: %w", err)
	}

	return transcodedPacket, nil
}

func (t *TranscodingService) getMixer(key string) *audiomultiplexer.Mixer {
	t.mu.Lock()
	defer t.mu.Unlock()

	if m, exists := t.mixers[key]; exists {
		return m
	}

	m := audiomultiplexer.NewMixer(8000, 20) // Assuming 8kHz sample rate and 20ms frame size
	t.mixers[key] = m
	return m
}

func (t *TranscodingService) rtpToSamples(packet *rtp.Packet, codecName string) ([]float32, error) {
	switch codecName {
	case "PCMU":
		return t.pcmuToSamples(packet.Payload)
	case "PCMA":
		return t.pcmaToSamples(packet.Payload)
	case "opus":
		return t.opusToSamples(packet.Payload)
	default:
		return nil, fmt.Errorf("unsupported codec for decoding: %s", codecName)
	}
}

func (t *TranscodingService) samplesToRTP(samples []float32, header rtp.Header, codecName string) (*rtp.Packet, error) {
	var payload []byte
	var err error

	switch codecName {
	case "PCMU":
		payload, err = t.samplesToPCMU(samples)
	case "PCMA":
		payload, err = t.samplesToPCMA(samples)
	case "opus":
		payload, err = t.samplesToOpus(samples)
	default:
		return nil, fmt.Errorf("unsupported codec for encoding: %s", codecName)
	}

	if err != nil {
		return nil, err
	}

	return &rtp.Packet{
		Header:  header,
		Payload: payload,
	}, nil
}

// Implement codec-specific conversion functions
func (t *TranscodingService) pcmuToSamples(payload []byte) ([]float32, error) {
	samples := make([]float32, len(payload))
	for i, b := range payload {
		samples[i] = float32(pcmuToLinear(b)) / 32768.0
	}
	return samples, nil
}

func (t *TranscodingService) pcmaToSamples(payload []byte) ([]float32, error) {
	// TODO: Implement PCMA to float32 samples conversion
	return nil, fmt.Errorf("PCMA to samples conversion not implemented")
}

func (t *TranscodingService) opusToSamples(payload []byte) ([]float32, error) {
	// TODO: Implement Opus to float32 samples conversion
	return nil, fmt.Errorf("Opus to samples conversion not implemented")
}

func (t *TranscodingService) samplesToPCMU(samples []float32) ([]byte, error) {
	payload := make([]byte, len(samples))
	for i, s := range samples {
		payload[i] = linearToPCMU(int16(s * 32768.0))
	}
	return payload, nil
}

func (t *TranscodingService) samplesToPCMA(samples []float32) ([]byte, error) {
	// TODO: Implement float32 samples to PCMA conversion
	return nil, fmt.Errorf("Samples to PCMA conversion not implemented")
}

func (t *TranscodingService) samplesToOpus(samples []float32) ([]byte, error) {
	// TODO: Implement float32 samples to Opus conversion
	return nil, fmt.Errorf("Samples to Opus conversion not implemented")
}

func pcmuToLinear(u uint8) int16 {
	u = ^u
	s := int16((u & 0x80) << 8)
	e := (u & 0x70) >> 4
	f := int16(u & 0x0f)
	m := (f << 3) + 0x84
	if e > 0 {
		m <<= (e - 1)
	} else {
		m >>= 1
	}
	return s | m
}

func linearToPCMU(sample int16) uint8 {
	// Constants for μ-law encoding
	const (
		BIAS = 0x84
		CLIP = 32635
		SIGN = 0x80
	)

	// Get the absolute value and apply bias
	var value uint16
	if sample < 0 {
		value = uint16(-sample)
		value += BIAS
	} else {
		value = uint16(sample)
		value = BIAS - value
	}

	// Clip the value
	if value > CLIP {
		value = CLIP
	}

	// Convert to 8-bit μ-law
	var exponent uint8 = 0
	for value > 255 {
		exponent++
		value >>= 1
	}

	var mantissa uint8 = uint8(value) & 0x7F
	var encoded uint8 = exponent<<4 | mantissa>>3

	// Apply sign bit and invert all bits
	if sample > 0 {
		encoded |= SIGN
	}
	return ^encoded
}
