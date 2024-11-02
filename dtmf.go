package sipnexus

import (
	"fmt"
	"sync"

	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type DTMFHandler struct {
	logger        logger.Logger
	dtmfPayload   uint8
	eventHandlers map[string]func(string)
	mu            sync.RWMutex
}

func NewDTMFHandler(logger logger.Logger, dtmfPayload uint8) *DTMFHandler {
	return &DTMFHandler{
		logger:        logger,
		dtmfPayload:   dtmfPayload,
		eventHandlers: map[string]func(string){},
	}
}

func (d *DTMFHandler) processEvent(event string) {
	d.logger.Info(fmt.Sprintf("DTMF event received: %s", event))
	d.notifyHandlers(event)
}

func (d *DTMFHandler) HandleDTMF(packet *rtp.Packet) error {
	if packet.PayloadType != d.dtmfPayload {
		return fmt.Errorf("not a DTMF event packet")
	}

	if len(packet.Payload) < 4 {
		return fmt.Errorf("invalid DTMF packet length")
	}

	event := packet.Payload[0]
	endOfEvent := (packet.Payload[1] & 0x80) != 0

	char := d.mapDTMFEventToChar(event)

	if endOfEvent {
		go d.processEvent(char)
	}

	return nil
}

func (d *DTMFHandler) mapDTMFEventToChar(event byte) string {
	switch event {
	case 0:
		return "0"
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	case 4:
		return "4"
	case 5:
		return "5"
	case 6:
		return "6"
	case 7:
		return "7"
	case 8:
		return "8"
	case 9:
		return "9"
	case 10:
		return "*"
	case 11:
		return "#"
	case 12:
		return "A"
	case 13:
		return "B"
	case 14:
		return "C"
	case 15:
		return "D"
	default:
		return "?"
	}
}

func (d *DTMFHandler) RegisterHandler(event string, handler func(string)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.eventHandlers[event] = handler
}

func (d *DTMFHandler) notifyHandlers(event string) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if handler, ok := d.eventHandlers[event]; ok {
		go handler(event)
	}
}

func (d *DTMFHandler) IntegrateWithPion(pc *webrtc.PeerConnection) {
	// TODO: need to find a way to get DTMF tone in a call
}

func (d *DTMFHandler) ProcessDTMFEvent(event string, sessionID string) {
	d.logger.Info(fmt.Sprintf("Processing DTMF event %s for session %s", event, sessionID))

	// TODO: Implement DTMF-triggered actions (e.g., IVR menu navigation)
	switch event {
	case "1":
		d.logger.Info("DTMF 1 pressed: Navigating to main menu")
	case "2":
		d.logger.Info("DTMF 2 pressed: Transferring to agent")
	default:
		d.logger.Info("Unhandled DTMF event: " + event)
	}

	d.notifyHandlers(event)
}
