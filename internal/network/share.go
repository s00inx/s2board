package network

// TODO: явно передавать логгер

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/s00inx/stdesk/internal/models"
)

/*
POST /api/recv - принять манифест
*/

func (n *Node) Broadcast(man *models.NoteManifest) {
	peers := n.GetConns()
	if len(peers) == 0 {
		log.Println("[BROADCAST] no peers found to broadcast.")
		return
	}

	jsonData, err := json.Marshal(man)
	if err != nil {
		log.Printf("[BROADCAST] marshal error: %v", err)
		return
	}

	log.Printf("[BROADCAST] sending update to %d peers...\n", len(peers))

	for _, p := range peers {
		go func(peer models.Peer) {
			url := fmt.Sprintf("http://%s:%d/api/recv", peer.IP, peer.Port)
			client := http.Client{Timeout: 5 * time.Second}

			resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("[BROADCAST] failed to send to %s: %v", peer.UID[:8], err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				log.Printf("[BROADCAST] delivered to %s", peer.UID[:8])
			}
		}(p)
	}
}
