package transport

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/s00inx/s2board/internal/codec"
	"github.com/s00inx/s2board/internal/models"
)

// RU: в mvp для транспорта был выбран именно HTTP потому что он прост в отладке, но влечет за собой оверхед
// на заголовки и json, также я использовал только Push на бекенде, Pull оставил только для фронтенда,
// это просто упрощение, в p2p сетях лучше использовать только Push!
// за всю п2п логику отвечает именно 1 эндпоинт /api/p2p, только POST (нет смысла следовать rest)
// todo: сделать бинарный протокол поверх udp

type HTTPTransport struct {
	Client http.Client
	Codec  codec.Codec

	Port int
}

// send a p2p packet to exact ip:port
func (t *HTTPTransport) Sendp(ip string, port int, packet2send *models.P2PPacket) (*models.P2PPacket, error) {
	// sending packet to addr
	data2s, err := t.Codec.Encode(packet2send)
	if err != nil {
		return nil, fmt.Errorf("")
	}

	resp, err := t.Client.Post(fmt.Sprintf("http://%s:%d/p2p", ip, port), "application/json", bytes.NewReader(data2s))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// process response packet
	var respp models.P2PPacket
	if err := t.Codec.DecodeStream(resp.Body, &respp); err != nil {
		return nil, err
	}

	if !respp.Verify() {
		return nil, fmt.Errorf("invalid signature from %s:%d", ip, port)
	}

	return &respp, nil
}

// broadcast P2PPacket to all known peers in local network
func (t *HTTPTransport) Broadcastp(pp *models.P2PPacket, ps []string) {
	p2send, err := t.Codec.Encode(pp)
	if err != nil {
		log.Printf("[broadcast] marshal error: %v -> ignored", err)
		return
	}

	log.Printf("[p2p] broadcasting to %d peers", len(ps))
	for _, dst := range ps {
		go func(p string) {
			var dsturl string = fmt.Sprintf("http://%s/p2p", p)
			resp, err := t.Client.Post(dsturl, "application/json", bytes.NewReader(p2send))
			if err != nil {
				log.Printf("[broadcast] failed to send to %s", p)
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("[broadcast] code %d from %s, packet: %v", resp.StatusCode, p, pp)
			} else {
				log.Printf("[broadcast] delivered to %s", p)
			}
		}(dst)
	}
}

// setup node as Server for receiving p2ppackets
func (t *HTTPTransport) Start(port int, handler func(p *models.P2PPacket, rmaddr string) (*models.P2PPacket, error)) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/p2p", func(w http.ResponseWriter, r *http.Request) {
		var incp models.P2PPacket
		if err := t.Codec.DecodeStream(r.Body, &incp); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp, err := handler(&incp, r.RemoteAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp != nil {
			data, _ := t.Codec.Encode(resp)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})

	log.Printf("[transport] HTTP server started on :%d", port)
	return mux
}
