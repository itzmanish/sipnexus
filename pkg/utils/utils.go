package utils

import (
	"fmt"
	"time"

	"github.com/pion/sdp/v3"
)

func ParseSDP(body []byte) (*sdp.SessionDescription, error) {
	var sd sdp.SessionDescription
	err := sd.Unmarshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SDP: %w", err)
	}
	return &sd, nil
}

func GenerateSDP(mediaTypes []string) (*sdp.SessionDescription, error) {
	sd, err := sdp.NewJSEPSessionDescription(false)
	if err != nil {
		return nil, err
	}
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

	return sd, nil
}
