package sipnexus

import (
	"fmt"
	"sync"

	muxgo "github.com/itzmanish/audio-multiplexer-go"
	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/pion/rtp"
)

type ConferenceRoom struct {
	ID           string
	Participants map[string]*Participant
	Mixer        *muxgo.Mixer
	mu           sync.RWMutex
}

type Participant struct {
	ID     string
	Stream chan *rtp.Packet
}

type ConferencingService struct {
	logger logger.Logger
	rooms  map[string]*ConferenceRoom
	mu     sync.RWMutex
}

func NewConferencingService(logger logger.Logger) *ConferencingService {
	return &ConferencingService{
		logger: logger,
		rooms:  make(map[string]*ConferenceRoom),
	}
}

func (c *ConferencingService) CreateRoom(roomID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.rooms[roomID]; exists {
		return fmt.Errorf("room %s already exists", roomID)
	}

	mixer := muxgo.NewMixer(8000, 20) // Assuming 8kHz sample rate and 20ms frame size
	c.rooms[roomID] = &ConferenceRoom{
		ID:           roomID,
		Participants: make(map[string]*Participant),
		Mixer:        mixer,
	}

	c.logger.Info(fmt.Sprintf("Created conference room: %s", roomID))
	return nil
}

func (c *ConferencingService) JoinRoom(roomID, participantID string) (chan *rtp.Packet, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	room, exists := c.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room %s does not exist", roomID)
	}

	if _, exists := room.Participants[participantID]; exists {
		return nil, fmt.Errorf("participant %s already in room %s", participantID, roomID)
	}

	stream := make(chan *rtp.Packet, 100)
	room.Participants[participantID] = &Participant{
		ID:     participantID,
		Stream: stream,
	}

	room.Mixer.AddSource(participantID)

	c.logger.Info(fmt.Sprintf("Participant %s joined room %s", participantID, roomID))
	return stream, nil
}

func (c *ConferencingService) LeaveRoom(roomID, participantID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	room, exists := c.rooms[roomID]
	if !exists {
		return fmt.Errorf("room %s does not exist", roomID)
	}

	if _, exists := room.Participants[participantID]; !exists {
		return fmt.Errorf("participant %s not in room %s", participantID, roomID)
	}

	close(room.Participants[participantID].Stream)
	delete(room.Participants, participantID)
	room.Mixer.RemoveSource(participantID)

	c.logger.Info(fmt.Sprintf("Participant %s left room %s", participantID, roomID))
	return nil
}

func (c *ConferencingService) ProcessRTPPacket(roomID string, participantID string, packet *rtp.Packet) {
	c.mu.RLock()
	room, exists := c.rooms[roomID]
	c.mu.RUnlock()

	if !exists {
		c.logger.Warn("Attempt to process RTP packet for non-existent room: " + roomID)
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	// Convert RTP packet payload to audio samples
	samples := c.rtpToSamples(packet)

	// Add samples to the mixer
	room.Mixer.AddSamples(participantID, samples)

	// Mix the audio
	mixedSamples := room.Mixer.Mix()

	// Convert mixed samples back to RTP packet
	mixedPacket := c.samplesToRTP(mixedSamples, packet.Header)

	// Send the mixed audio to all participants except the sender
	for id, participant := range room.Participants {
		if id != participantID {
			select {
			case participant.Stream <- mixedPacket:
			default:
				c.logger.Warn("Participant buffer full, dropping packet: " + id)
			}
		}
	}
}

func (c *ConferencingService) rtpToSamples(packet *rtp.Packet) []float32 {
	samples := make([]float32, len(packet.Payload))
	for i, b := range packet.Payload {
		samples[i] = float32(int16(b<<8)) / 32768.0
	}
	return samples
}

func (c *ConferencingService) samplesToRTP(samples []float32, header rtp.Header) *rtp.Packet {
	payload := make([]byte, len(samples))
	for i, s := range samples {
		payload[i] = byte(int16(s*32768.0) >> 8)
	}
	return &rtp.Packet{
		Header:  header,
		Payload: payload,
	}
}
