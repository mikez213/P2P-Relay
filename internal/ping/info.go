package customprotocol

import (
	"fmt"

	p2p "mnwarm/internal/ping/pb"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	proto "google.golang.org/protobuf/proto"
	// "github.com/elastic/go-sysinfo"
)

type InfoRequestHandler struct {
	protocol *PingProtocol
}

func (h *InfoRequestHandler) Handle(s network.Stream, from peer.ID, data []byte) error {
	var req p2p.InfoRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		log.Errorf("Failed to unmarshal InfoRequest: %v", err)
		s.Reset()
		return err
	}

	log.Infof("Received InfoRequest from %s: HostID=%s",
		from, req.HostId)

	log.Infof("Our addrs %s", h.protocol.host.Addrs())
	publicIP := "0.0.0.0"
	privateIP := "192.0.2.1"
	isPublic := false
	systemConfig := map[string]string{
		"os":     "macOS",
		"arch":   "amd64",
		"uptime": "null",
	}

	resp := &p2p.InfoResponse{
		HostId:        h.protocol.host.ID().String(),
		PublicIp:      publicIP,
		PrivateIp:     privateIP,
		IsPublic:      isPublic,
		ClientVersion: clientVersion,
		SystemConfig:  systemConfig,
	}

	ok := h.protocol.sendProtoMessage(s.Conn().RemotePeer(), infoResponse, resp)

	if ok {
		log.Infof("%s: InfoResponse sent to %s.", h.protocol.host.ID().String(), from.String())
	} else {
		err := fmt.Errorf("%s: Error in sending InfoResponse to %s", h.protocol.host.ID().String(), from.String())
		return err
	}

	log.Infof("Sent InfoResponse to %s: HostID=%s, PublicIP=%s", from, resp.HostId, resp.PublicIp)
	h.protocol.done <- true
	return nil
}

type InfoResponseHandler struct {
	protocol *PingProtocol
}

func (h *InfoResponseHandler) Handle(s network.Stream, from peer.ID, data []byte) error {
	var resp p2p.InfoResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		log.Errorf("Failed to unmarshal InfoResponse: %v", err)
		s.Reset()
		return err
	}

	log.Infof("Received InfoResponse from %s: HostID=%s, PublicIP=%s, PrivateIP=%s, IsPublic=%v, ClientVersion=%s, SystemConfig=%v",
		from, resp.HostId, resp.PublicIp, resp.PrivateIp, resp.IsPublic, resp.ClientVersion, resp.SystemConfig)
	h.protocol.done <- true
	return nil
}
