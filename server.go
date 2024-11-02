package sipnexus

import (
	"context"
	"fmt"
	"sync"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/itzmanish/sipnexus/pkg/utils"
)

type Server struct {
	logger         logger.Logger
	srv            *sipgo.Server
	instances      []string
	hashRing       *ConsistentHash
	mu             sync.RWMutex
	sessionManager *SessionManager
}

func NewServer(log logger.Logger) (*Server, error) {
	s := &Server{
		logger:         log,
		instances:      []string{"instance1", "instance2", "instance3"},
		sessionManager: NewSessionManager(),
	}

	// Initialize consistent hash ring
	s.hashRing = NewConsistentHash(100) // 100 virtual nodes per instance
	for _, instance := range s.instances {
		s.hashRing.Add(instance)
	}

	ua, err := sipgo.NewUA(sipgo.WithUserAgent("sipnexus"))
	if err != nil {
		return nil, fmt.Errorf("failed to create UA: %w", err)
	}
	// Create a new SIP server
	srv, err := sipgo.NewServer(ua, sipgo.WithServerLogger(log.(*logger.ZeroLogger).InternalLogger()))
	if err != nil {
		return nil, fmt.Errorf("failed to create SIP server: %w", err)
	}

	// srv.ServeRequest(func(r *sip.Request) {
	// 	callID := r.CallID()
	// 	instance := s.getInstanceForRequest(callID.String())

	// 	if instance != "instance1" { // Assume this is instance1
	// 		s.logger.Info("Request routed to different instance: " + instance)
	// 		return
	// 	}
	// })

	srv.OnAck(s.handleAck)
	srv.OnOptions(s.handleOptions)
	srv.OnInvite(s.handleInvite)
	srv.OnBye(s.handleBye)
	srv.OnCancel(s.handleCancel)
	srv.OnRegister(s.handleRegister)

	s.srv = srv
	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting SIP server...")
	return s.srv.ListenAndServe(ctx, "udp", "0.0.0.0:5060")
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down SIP server...")
	return s.srv.Close()
}

func (s *Server) getInstanceForRequest(callID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hashRing.Get(callID)
}

func (s *Server) handleInvite(req *sip.Request, tx sip.ServerTransaction) {
	s.logger.Infof("handling invite request: %v", req)
	// Create a new session
	session := s.sessionManager.CreateSession(req.CallID().String())

	// Parse SDP offer
	_, err := utils.ParseSDP(req.Body())
	if err != nil {
		s.sendErrorResponse(req, tx, sip.StatusBadRequest, "Bad Request: Invalid SDP")
		return
	}

	// Handle media setup
	answerSDP, err := session.rtc.SetOffer(string(req.Body()))
	if err != nil {
		s.logger.Errorf("failed to generate answer: %v", err)
		s.sendErrorResponse(req, tx, sip.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Create 200 OK response with SDP answer
	resp := sip.NewResponseFromRequest(req, 200, "OK", []byte(answerSDP))
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response: " + err.Error())
	}
}

func (s *Server) handleAck(req *sip.Request, tx sip.ServerTransaction) {
	// ACK doesn't require a response, but we can use it to finalize the call setup
	s.logger.Info("Received ACK for call: " + req.CallID().String())

}

func (s *Server) handleOptions(req *sip.Request, tx sip.ServerTransaction) {
	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	tx.Respond(res)
}

func (s *Server) handleBye(req *sip.Request, tx sip.ServerTransaction) {
	// End the session
	callId := req.CallID()
	if callId == nil {
		s.logger.Error("call id is empty, not handling bye")
	}

	s.sessionManager.DeleteSession(callId.Value())

	// Send 200 OK response
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response: " + err.Error())
	}
}

func (s *Server) handleCancel(req *sip.Request, tx sip.ServerTransaction) {
	// Handle CANCEL request
	s.logger.Infof("Received CANCEL for call: %v", req.CallID())

	// Send 200 OK for the CANCEL request
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response for CANCEL: " + err.Error())
	}
	tx.Terminate()
	// TODO: Terminate the pending INVITE transaction if it exists
}

func (s *Server) handleRegister(req *sip.Request, tx sip.ServerTransaction) {
	authHeader := req.GetHeader("Authorization")
	if authHeader == nil || !s.isValidToken(authHeader.Value()) {
		s.sendErrorResponse(req, tx, sip.StatusUnauthorized, "Unauthorized")
		return
	}

	// Proceed with registration logic
	s.logger.Info("Received REGISTER request")
	resp := sip.NewResponseFromRequest(req, 200, "OK", nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error("Failed to send 200 OK response for REGISTER: " + err.Error())
	}
}

func (s *Server) sendErrorResponse(req *sip.Request, tx sip.ServerTransaction, statusCode sip.StatusCode, reason string) {
	s.logger.Errorf("returning error: %v", reason)
	resp := sip.NewResponseFromRequest(req, statusCode, reason, nil)
	if err := tx.Respond(resp); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to send %d response: %s", statusCode, err.Error()))
	}
}

func (s *Server) isValidToken(token string) bool {
	// Implement token validation logic here
	return token == "valid-token" // Placeholder for actual validation
}
