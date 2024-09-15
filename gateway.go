package sipnexus

import (
	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type MediaGateway struct {
	logger             logger.Logger
	transcodingService *TranscodingService
	peerConnections    map[string]*webrtc.PeerConnection
}

func NewMediaGateway(logger logger.Logger) *MediaGateway {
	return &MediaGateway{
		logger:          logger,
		peerConnections: make(map[string]*webrtc.PeerConnection),
	}
}

func (g *MediaGateway) HandleMediaSetup(offer string, sessionID string) (string, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return "", err
	}

	// Set up handlers for ICE and media tracks
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		g.logger.Info("ICE Connection State has changed: " + connectionState.String())
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		g.handleIncomingTrack(track, sessionID)
	})

	err = peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  offer,
		Type: webrtc.SDPTypeOffer,
	})
	if err != nil {
		return "", err
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return "", err
	}

	g.peerConnections[sessionID] = peerConnection

	return answer.SDP, nil
}

func (g *MediaGateway) handleIncomingTrack(track *webrtc.TrackRemote, sessionID string) {
	g.logger.Info("New incoming track")

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			g.logger.Error("Error reading RTP packet: " + err.Error())
			return
		}

		// Process the RTP packet (e.g., transcoding, forwarding to conference)
		g.processRTPPacket(rtpPacket, track.Codec().MimeType, sessionID)
	}
}

func (g *MediaGateway) processRTPPacket(packet *rtp.Packet, codecMimeType string, sessionID string) {
	// Determine if transcoding is needed
	targetCodec := "opus" // Example target codec

	if codecMimeType != targetCodec {
		transcodedPacket, err := g.transcodingService.TranscodePacket(packet, codecMimeType, targetCodec)
		if err != nil {
			g.logger.Error("Error transcoding packet: " + err.Error())
			return
		}
		packet = transcodedPacket
	}

	// TODO: Forward the packet to the appropriate destination (e.g., conference room)
}

func (g *MediaGateway) SetTranscodingService(ts *TranscodingService) {
	g.transcodingService = ts
}
