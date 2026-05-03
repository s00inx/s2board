// p2p packet crypto and data encapsulation logic
package models

import (
	"crypto/ed25519"
	"encoding/hex"
)

// RU: единая структура пакета позволяет инкапсулировать его в ЛЮБОЙ протокол, в том числе бинарный.
//     в качестве упрощения для MVP был выбран http.

// unified broadcast packet above tcp for scalable file sharing
type P2PPacket struct {
	Action    Actcode `json:"a"`
	Senderuid string  `json:"s"`
	Payload   []byte  `json:"p"`
	Signature string  `json:"si"`
}

// build a new packet for broadcast
func NewPacket(payload []byte, action Actcode, nodeuid string, privk ed25519.PrivateKey) *P2PPacket {
	pk := P2PPacket{
		Senderuid: nodeuid,
		Payload:   payload,
		Action:    Actcode(action),
	}

	// sign: man json + action
	hdata := append(payload, byte(action))
	pk.Signature = hex.EncodeToString(ed25519.Sign(privk, hdata))

	return &pk
}

// verify packet (authencity and integrity)
func (p *P2PPacket) Verify() bool {
	pub, err := hex.DecodeString(p.Senderuid)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}

	sig, err := hex.DecodeString(p.Signature)
	if err != nil {
		return false
	}

	hdata := append(p.Payload, byte(p.Action))
	return ed25519.Verify(pub, hdata, sig)
}
