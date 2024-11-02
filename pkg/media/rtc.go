package media

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"

	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/pion/sdp/v3"
)

var SupportedCodecs = []string{"pcmu", "PCMU"}

type MediaEngine interface {
	SetOffer(offer string) (string, error)
}

type UDPMediaEngine struct {
	logger        logger.Logger
	rtpConn       *UDPConn
	selectedCodec []string // [format,codecName, clockRate], [8,pcma, 8000]
	selectedCI    string   // selected connection information
}

func NewUDPMediaEngine(log logger.Logger) MediaEngine {
	me := &UDPMediaEngine{
		logger: log,
	}

	return me
}

func (ume *UDPMediaEngine) SetOffer(offer string) (string, error) {
	var sd sdp.SessionDescription
	err := sd.Unmarshal([]byte(offer))
	if err != nil {
		return "", fmt.Errorf("failed to parse SDP: %w", err)
	}

	err = ume.validateFormats(sd.MediaDescriptions)
	if err != nil {
		return "", err
	}
	conn, err := NewUDPConn(net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: getFreePort(),
	})
	if err != nil {
		return "", err
	}
	ume.rtpConn = conn
	ume.logger.Infof("ci: %v", ume.selectedCI)
	err = ume.rtpConn.SetRemoteAddr(net.UDPAddrFromAddrPort(netip.MustParseAddrPort(ume.selectedCI)))
	if err != nil {
		return "", err
	}
	go ume.readLoop()
	return ume.generateAnswer(sd)
}

func (ume *UDPMediaEngine) generateAnswer(offer sdp.SessionDescription) (string, error) {
	answer := sdp.SessionDescription{
		Version: 0,
		Origin: sdp.Origin{
			Username:       "-",
			SessionID:      offer.Origin.SessionID,
			SessionVersion: offer.Origin.SessionID + 2,
			NetworkType:    "IN",
			AddressType:    "IP4",
			UnicastAddress: "0.0.0.0",
		},
		SessionName: "SIP Nexus",
		ConnectionInformation: &sdp.ConnectionInformation{
			NetworkType: "IN",
			AddressType: "IP4",
			Address:     &sdp.Address{Address: "0.0.0.0"},
		},
		TimeDescriptions: []sdp.TimeDescription{
			{
				Timing: sdp.Timing{
					StartTime: 0,
					StopTime:  0,
				},
			},
		},
		MediaDescriptions: []*sdp.MediaDescription{
			{
				MediaName: sdp.MediaName{
					Media:   "audio",
					Port:    sdp.RangedPort{Value: ume.rtpConn.localAddr.Port},
					Protos:  []string{"RTP", "AVP"},
					Formats: []string{"0"},
				},
				Attributes: []sdp.Attribute{
					{Key: "rtpmap", Value: "0 PCMU/8000"},
					{Key: "ptime", Value: "20"},
					{Key: "maxptime", Value: "150"},
					{Key: "sendrecv"},
				},
			},
		}}
	ans, err := answer.Marshal()
	return string(ans), err
}

func (ume *UDPMediaEngine) validateFormats(mediaDesc []*sdp.MediaDescription) error {
	// 111 - opus, 0 - pcmu, 8 - pcma
	filterSupportedCodec := func(codec string) (string, string, bool) {
		values := strings.Split(codec, "/")
		if len(values) < 2 {
			return "", "", false
		}
		codecName := values[0]
		clockRate := values[1]
		return codecName, clockRate, slices.Contains(SupportedCodecs, codecName)
	}
	validCodecs := [][]string{}
	for _, md := range mediaDesc {
		if md.MediaName.Media != "audio" {
			continue
		}
		for _, a := range md.Attributes {
			if a.Key != "rtpmap" {
				continue
			}
			values := strings.Split(a.Value, " ")
			if len(values) != 2 {
				continue
			}
			format := values[0]
			codec := values[1]
			ume.logger.Infof("got codec info %#v", values)
			if codecName, rate, ok := filterSupportedCodec(codec); ok {
				validCodecs = append(validCodecs, []string{format, codecName, rate})
				if ume.selectedCI == "" {
					ume.selectedCI = fmt.Sprintf("%v:%v", md.ConnectionInformation.Address.String(), md.MediaName.Port.Value)
				}
			}
		}
	}
	if len(validCodecs) == 0 {
		return errors.New("invalid codecs")
	}

	ume.selectedCodec = validCodecs[0]

	return nil
}

// TODO: implement close as well
func (ume *UDPMediaEngine) readLoop() {
	for {
		buf := make([]byte, 1024)
		n, _, err := ume.rtpConn.conn.ReadFrom(buf)
		if err != nil {
			return
		}
		ume.logger.Infof("read packets: %v", n)
	}
}

type UDPConn struct {
	conn       *net.UDPConn
	localAddr  net.UDPAddr
	remoteAddr *net.UDPAddr
}

func getFreePort() int {
	return 10001
}

func NewUDPConn(laddr net.UDPAddr) (*UDPConn, error) {
	conn, err := net.ListenUDP("udp", &laddr)
	if err != nil {
		return nil, err
	}
	c := &UDPConn{
		conn:      conn,
		localAddr: laddr,
	}
	return c, err
}

func (uc *UDPConn) SetRemoteAddr(addr *net.UDPAddr) error {
	if uc.remoteAddr != nil {
		return errors.New("remote addr is already set")
	}
	uc.remoteAddr = addr
	return nil
}
