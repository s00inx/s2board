package transport

import (
	"net/http"

	"github.com/s00inx/s2board/internal/models"
)

type Transport interface {
	// send p2p packet to ip:port
	Sendp(ip string, port int, packet2send *models.P2PPacket) (*models.P2PPacket, error)
	// send p2p packet to all peers in local network
	Broadcastp(packet2bc *models.P2PPacket, ps []string)
	// start node dispatcher
	Start(port int, handler func(p *models.P2PPacket) (*models.P2PPacket, error)) *http.ServeMux
}
