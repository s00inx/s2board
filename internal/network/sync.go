package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/s00inx/stdesk/internal/models"
)

// GET /api/sync - список всех хешей которые есть у конкретной ноды
func (n *Node) GetHashes(w http.ResponseWriter, r *http.Request) {
	hashes, err := n.Storage.GetHashes()
	if err != nil {
		http.Error(w, "Internal error", 500)
		return
	}
	json.NewEncoder(w).Encode(hashes)
}

// POST /api/sync/fetch - принимает список хешей и отдает полные манифесты
func (n *Node) FetchManifests(w http.ResponseWriter, r *http.Request) {
	var re []string
	if err := json.NewDecoder(r.Body).Decode(&re); err != nil {
		http.Error(w, "Bad request", 400)
		return
	}

	var res []models.NoteManifest
	for _, h := range re {
		man, err := n.Storage.GetManifest(h)
		if err == nil && man != nil {
			res = append(res, *man)
		}
	}
	json.NewEncoder(w).Encode(res)
}

// синкануть 2 пира между собой
func (n *Node) Syncw(p models.Peer) error {
	c := &http.Client{Timeout: 10 * time.Second}

	resp, err := c.Get(fmt.Sprintf("http://%s:%d/api/sync", p.IP, p.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var rmhashes []string
	if err := json.NewDecoder(resp.Body).Decode(&rmhashes); err != nil {
		return err
	}

	var ms []string
	for _, h := range rmhashes {
		_, err := n.Storage.GetManifest(h)
		if err != nil {
			ms = append(ms, h)
		}
	}

	if len(ms) == 0 {
		return nil
	}
	log.Printf("[SYNC] found %d missing notes from %s, fetching...", len(ms), p.UID[:8])

	body, _ := json.Marshal(ms)
	fresp, err := c.Post(
		fmt.Sprintf("http://%s:%d/api/sync/fetch", p.IP, p.Port),
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return err
	}
	defer fresp.Body.Close()

	var newnotes []models.NoteManifest
	if err := json.NewDecoder(fresp.Body).Decode(&newnotes); err != nil {
		return err
	}

	for _, man := range newnotes {
		if !man.Verify() {
			log.Printf("[WARNING] fake signature for note %s from peer %s --> ignored.", man.Hash, p.UID[:8])
			continue
		}

		n.Storage.SaveFile(man)

		if man.FileHash != "" && !n.Storage.FileExists(man.FileHash) {
			log.Printf("[SYNC] downloading blob for note: %s", man.Title)
			err := n.DlBlob(p, man.FileHash)
			if err != nil {
				log.Printf("[ERR] failed to download blob %s: %v", man.FileHash[:8], err)
			}
		}

		err = n.Storage.SaveFile(man)
		if err == nil {
			log.Printf("[SYNC] added new note: %s", man.Title)
		}
	}

	return nil
}
