package sipnexus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/pion/sdp/v3"
)

type Server struct {
	logger              logger.Logger
	srv                 *sipgo.Server
	instances           []string
	hashRing            *ConsistentHash
	mu                  sync.RWMutex
	mediaGateway        *MediaGateway
	transcodingService  *TranscodingService
	conferencingService *ConferencingService
	dtmfHandler         *DTMFHandler
}

func NewServer(logger logger.Logger) (*Server, error) {
	s := &Server{
		logger:              logger,
		instances:           []string{"instance1", "instance2", "instance3"},
		mediaGateway:        NewMediaGateway(logger),
		transcodingService:  NewTranscodingService(logger),
		conferencingService: NewConferencingService(logger),
		dtmfHandler:         NewDTMFHandler(logger, 101), // Assuming 101 is the DTMF payload type
	}

	// Initialize consistent hash ring
	s.hashRing = NewConsistentHash(100) // 100 virtual nodes per instance
	for _, instance := range s.instances {
		s.hashRing.Add(instance)
	}

	// Create a new SIP server
	srv, err := sipgo.NewServer(sipgo.ServerConfig{
		Host:           "0.0.0.0",
		Port:           5060,
		Handler:        s,
		Network:        "udp",
		Logger:         logger,
		AllowedPeers:   []string{"0.0.0.0/0"},
		SipDomain:      "example.com",
		AllowedDomains: []string{"example.com"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SIP server: %w", err)
	}

	s.srv = srv
	return s, nil
}

func (s *Server) Start() error {
	s.logger.Info("Starting SIP server...")
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down SIP server...")
	return s.srv.Shutdown(ctx)
}

func (s *Server) ServeRequest(req *sip.Request, tx sip.ServerTransaction) {
	method := req.Method()
	s.logger.Info("Received SIP request: " + method)

	callID := req.CallID()
	instance := s.getInstanceForRequest(callID)

	if instance != "instance1" { // Assume this is instance1
		s.logger.Info("Request routed to different instance: " + instance)
		return
	}

	switch method {
	case sip.INVITE:
		s.handleInvite(req, tx)
	case sip.ACK:
		s.handleAck(req, tx)
	case sip.BYE:
		s.handleBye(req, tx)
	case sip.CANCEL:
		s.handleCancel(req, tx)
	case sip.REGISTER:
		s.handleRegister(req, tx)
	default:
		s.handleUnsupportedMethod(req, tx)
	}
}

func (s *Server) getInstanceForRequest(callID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hashRing.Get(callID)
}

func (s *Server) handleInvite(req *sip.Request, tx sip.ServerTransaction) {
	// Create a new session
	session := s.sessionManager.CreateSession(req.CallID())

	// Parse SDP offer
	offerSDP, err := s.parseSDP(req.Body())
	if err != nil {
		s.sendErrorResponse(tx, 400, "Bad Request: Invalid SDP")
		return
	}

	// Handle media setup
	answerSDP, err := s.mediaGateway.HandleMediaSetup(offerSDP, session.ID)
	if err != nil {
		s.sendErrorResponse(tx, 500, "Internal Server Error")
		return
	}

	// Marshal the SDP answer
	answerBytes, err := answerSDP.Marshal()
	if err != nil {
		s.sendErrorResponse(tx, 500, "Internal Server Error")
		return
	}

	// Create 200 OK response with SDP answer
	resp := sip.NewResponseFromRequest(req, 200, "OK", answerBytes)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response: " + err.Error())
	}
}

func (s *Server) handleAck(req *sip.Request, tx sip.ServerTransaction) {
	// ACK doesn't require a response, but we can use it to finalize the call setup
	s.logger.Info("Received ACK for call: " + req.CallID())
}

func (s *Server) handleBye(req *sip.Request, tx sip.ServerTransaction) {
	// End the session
	s.sessionManager.DeleteSession(req.CallID())

	// Send 200 OK response
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response: " + err.Error())
	}
}

func (s *Server) handleCancel(req *sip.Request, tx sip.ServerTransaction) {
	// Handle CANCEL request
	s.logger.Info("Received CANCEL for call: " + req.CallID())

	// Send 200 OK for the CANCEL request
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response for CANCEL: " + err.Error())
	}

	// TODO: Terminate the pending INVITE transaction if it exists
}

func (s *Server) handleRegister(req *sip.Request, tx sip.ServerTransaction) {
	authHeader := req.Header.Get("Authorization")
	if !s.isValidToken(authHeader) {
		s.sendErrorResponse(tx, 401, "Unauthorized")
		return
	}

	// Proceed with registration logic
	s.logger.Info("Received REGISTER request")
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response for REGISTER: " + err.Error())
	}
}

func (s *Server) sendErrorResponse(tx sip.ServerTransaction, statusCode int, reason string) {
	resp := sip.NewResponseFromRequest(tx.GetRequest(), statusCode, reason, nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to send %d response: %s", statusCode, err.Error()))
	}
}

func (s *Server) parseSDP(body []byte) (*sdp.SessionDescription, error) {
	var sd sdp.SessionDescription
	err := sd.Unmarshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SDP: %w", err)
	}
	return &sd, nil
}

func (s *Server) generateSDP(sessionID string, mediaTypes []string) (*sdp.SessionDescription, error) {
	sd := sdp.NewJSEPSessionDescription(false)
	sd.Origin = sdp.Origin{
		Username:       "-",
		SessionID:      uint64(time.Now().Unix()),
		SessionVersion: uint64(time.Now().Unix()),
		NetworkType:    "IN",
		AddressType:    "IP4",
		UnicastAddress: "0.0.0.0",
	}
	sd.SessionName = "SIP Nexus Session"

	for _, mediaType := range mediaTypes {
		md := &sdp.MediaDescription{
			MediaName: sdp.MediaName{
				Media:   mediaType,
				Port:    sdp.RangedPort{Value: 9},
				Protos:  []string{"RTP", "AVP"},
				Formats: []string{"0"}, // Assuming PCM-u as default
			},
			ConnectionInformation: &sdp.ConnectionInformation{
				NetworkType: "IN",
				AddressType: "IP4",
				Address:     &sdp.Address{Address: "0.0.0.0"},
			},
		}
		sd.MediaDescriptions = append(sd.MediaDescriptions, md)
	}

	return &sd, nil
}

func (s *Server) isValidToken(token string) bool {
	// Implement token validation logic here
	return token == "valid-token" // Placeholder for actual validation
}
