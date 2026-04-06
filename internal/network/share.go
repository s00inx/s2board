package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

// раздать файл всем кто находится в одной локальной сети
func (n *Node) Broadcast(man *models.NoteManifest) {
	ps := n.GetConns()

	if len(ps) == 0 {
		log.Println("[BROADCAST] no peers found to broadcast.")
		return
	}

	jsond, err := json.Marshal(man)
	if err != nil {
		log.Printf("[BROADCAST] marshal error: %v", err)
		return
	}

	log.Printf("[BROADCAST] sending update to %d peers...\n", len(ps))
	for _, p := range ps {
		fmt.Println(p)
		go func(peer models.Peer) {
			c := http.Client{Timeout: 5 * time.Second}
			resp, err := c.Post(fmt.Sprintf("http://%s:%d/api/recv", peer.IP, peer.Port), "application/json", bytes.NewBuffer(jsond))
			if err != nil {
				log.Printf("[BROADCAST] failed to send to %s: %v", peer.UID[:8], err)
				return
			}
			defer resp.Body.Close()

			fmt.Println(resp.StatusCode)
			if resp.StatusCode == http.StatusOK {
				log.Printf("[BROADCAST] delivered to %s", peer.UID[:8])
			}
		}(p)
	}
}
