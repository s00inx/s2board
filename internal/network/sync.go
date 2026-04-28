// all sync nodes in local network logic
// hello and bye packets
package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/s00inx/s2board/internal/models"
)

// get list if all hashes
func (n *Node) GetHashes() ([]string, error) {
	return n.DbStorage.GetHashesList()
}

// принимает список хешей и отдает полные манифесты
func (n *Node) FetchManifests(hashes []string) ([]models.Manifest, error) {
	var res []models.Manifest

	if len(hashes) > 100 {
		hashes = hashes[:100]
	}

	for _, h := range hashes {
		man, err := n.DbStorage.GetManh(h, models.Bucketvirtual)
		if err == nil && man != nil {
			res = append(res, *man)
		}
	}

	return res, nil
}

// symmetric sync node and EXACT peer (args: peer and hash list)
func (n *Node) Syncw(p Peer, hl2send []string) {
	jd2send, err := json.Marshal(hl2send)
	if err != nil {
		return
	}

	hellop := models.NewBCp(jd2send, models.Acthello, n.UID, n.PrivateK)

	resp, err := n.client.Get(fmt.Sprintf("http://%s:%d/api/hello", p.IP, p.Port))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var rem []string
	if err := json.NewDecoder(resp.Body).Decode(&rem); err != nil {
		return
	}
	var missing []string
	for _, h := range rem {
		if !n.DbStorage.NoteExist(h) {
			missing = append(missing, h)
		}
	}

	// if nothing is missed - end
	if len(missing) == 0 {
		log.Printf("[sync] peer %s -> nothing to sync", p.UID[:8])
		return
	}

	log.Printf("[sync] peer %s: missing %d -> fetching", p.UID[:8], len(missing))
	body, _ := json.Marshal(missing)

	// do request with all missing hashes
	fresp, err := n.client.Post(
		fmt.Sprintf("http://%s:%d/api/fetch", p.IP, p.Port),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return
	}
	defer fresp.Body.Close()

	var newmanl []models.Manifest
	if err := json.NewDecoder(fresp.Body).Decode(&newmanl); err != nil {
		return
	}

	for _, man := range newmanl {
		if !man.Verify() {
			log.Printf("[sync] fake signature for note '%s' from peer %s -> ignored.", man.Title, p.UID[:8])
			continue
		}

		if man.FileSize == 0 && !n.DbStorage.NoteExist(man.Hash) {
			if err := n.DbStorage.Save2db(man, models.Bucketlocal); err != nil {
				return
			}
		}

		if man.FileSize > 0 && !n.FileStorage.FileExists(man.FileHash) && man.FileSize < mindlsize {
			err := n.DlFile(p, man.FileHash)

			if err != nil {
				log.Printf("[sync] failed to dl %s: %v -> ignored", man.FileHash[:8], err)
			}
			log.Printf("[sync] dl blob for note: %s -> ok", man.Title)
			if err := n.DbStorage.Save2db(man, models.Bucketlocal); err != nil {
				return
			}
		} else {
			n.filepeers.add(man.Hash, p)
		}

		if err := n.DbStorage.Save2db(man, models.Bucketvirtual); err != nil {
			return
		}
	}

	log.Printf("[sync] %s (%s) -> synced", p.Name, p.UID[:8])
}
