// синхронизация конкретной ноды с другими в этой же локальной сети
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

// GET /api/sync
// список всех хешей которые есть у конкретной ноды (нужно для синхронизации данных)
func (n *Node) GetHashes(w http.ResponseWriter, r *http.Request) {
	hashes, err := n.Storage.GetHashes()
	if err != nil {
		http.Error(w, "Internal error", 500)
		return
	}
	json.NewEncoder(w).Encode(hashes)
}

// POST /api/sync/fetch
// принимает список хешей и отдает полные манифесты
// наш узел просит у других те манифесты, которых не хватает
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

// полностью синхронизировать 2 ноды между собой в несколько этапов
func (n *Node) Syncw(p models.Peer) error {
	c := &http.Client{Timeout: 10 * time.Second}

	// 1: спрашиваем у другой ноды конкретный список ее пиров
	peresp, err := c.Get(fmt.Sprintf("http://%s:%d/api/peers", p.IP, p.Port))
	if err == nil {
		defer peresp.Body.Close()
		var ne []models.Peer
		if err := json.NewDecoder(peresp.Body).Decode(&ne); err == nil {
			for _, peer := range ne {
				if peer.UID != n.UID {
					n.peers.Add(peer)
				}
			}
		}
	}

	// 2:
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

	// 3:
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
