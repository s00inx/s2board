package internal

import (
	"crypto/ed25519"
	"fmt"
)

type GPacket struct {
	Data []byte // BoardEntry
	Sign []byte
	SUID string
}

func (n *NodeID) enterBroadcast(entry []byte) {
	packet := GPacket{
		Data: entry,
		SUID: n.UID,
	}

	packet.Sign = ed25519.Sign(n.PrivateK, []byte(entry))
	// 												^-- HASH

	for _, peer := range ActiveConns.Range {
		go func(p Peer) {
			_ = fmt.Sprintf("http://%s:%d/api/brc", p.IP, p.Port)

			// sendJson(url, packet)
		}(peer.(Peer))
	}
}
