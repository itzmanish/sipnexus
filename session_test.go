package sipnexus

import (
	"log"
	"testing"

	"github.com/pion/sdp/v3"
)

func TestSDPParsing(t *testing.T) {
	sdpString := `v=0
o=- 5928584521724734167 2 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE 0
a=extmap-allow-mixed
a=msid-semantic: WMS
m=audio 9 UDP/TLS/RTP/SAVPF 111 63 9 0 8 13 110 126
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:UCXx
a=ice-pwd:Qz8e93kPyHqlKDS6KuEPLYH/
a=ice-options:trickle
a=fingerprint:sha-256 CD:8F:13:4C:B1:65:41:AF:D4:FF:2F:67:AA:99:D2:70:4F:A4:6E:3A:86:3E:C4:F0:42:59:43:4E:7D:69:1D:54
a=setup:actpass
a=mid:0
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=extmap:2 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:3 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01
a=extmap:4 urn:ietf:params:rtp-hdrext:sdes:mid
a=sendrecv
a=msid:- 0a572566-fd9d-4db2-9f68-2744a2204ec1
a=rtcp-mux
a=rtcp-rsize
a=rtpmap:111 opus/48000/2
a=rtcp-fb:111 transport-cc
a=fmtp:111 minptime=10;useinbandfec=1
a=rtpmap:63 red/48000/2
a=fmtp:63 111/111
a=rtpmap:9 G722/8000
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:13 CN/8000
a=rtpmap:110 telephone-event/48000
a=rtpmap:126 telephone-event/8000
a=ssrc:4246817614 cname:ZYqE263ohLZ+0Ubp
a=ssrc:4246817614 msid:- 0a572566-fd9d-4db2-9f68-2744a2204ec1
`
	parsed := &sdp.SessionDescription{}
	parsed.Unmarshal([]byte(sdpString))

	log.Printf("parsed: %+v \n", parsed)
	for _, m := range parsed.MediaDescriptions {
		log.Printf("md: %#v \n", m)
	}
}
