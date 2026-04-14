// синхронизация конкретной ноды с другими в этой же локальной сети
// получить ВСЕ хеши -> сравнить -> запросить манифесты -> проверить -> сохранить
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
	return n.Storage.GetHashesList()
}

// принимает список хешей и отдает полные манифесты
func (n *Node) FetchManifests(hashes []string) ([]models.Manifest, error) {
	var res []models.Manifest

	if len(hashes) > 100 {
		hashes = hashes[:100]
	}

	for _, h := range hashes {
		man, err := n.Storage.Getmanh(h, models.Bucketvirtual)
		if err == nil && man != nil {
			res = append(res, *man)
		}
	}

	return res, nil
}

// sync node and peer
func (n *Node) Syncw(p models.Peer) error {
	// ask peer about its 'local' hashes
	resp, err := n.client.Get(fmt.Sprintf("http://%s:%d/api/hello", p.IP, p.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var rem []string
	if err := json.NewDecoder(resp.Body).Decode(&rem); err != nil {
		return err
	}
	var missing []string
	for _, h := range rem {
		if !n.Storage.HasNote(h) {
			missing = append(missing, h)
		}
	}

	// if nothing is missed - end
	if len(missing) == 0 {
		log.Printf("[sync] peer %s -> nothing to sync", p.UID[:8])
		return nil
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
		return err
	}
	defer fresp.Body.Close()

	var newmanl []models.Manifest
	if err := json.NewDecoder(fresp.Body).Decode(&newmanl); err != nil {
		return err
	}

	for _, man := range newmanl {
		if !man.Verify() {
			log.Printf("[sync] fake signature for note '%s' from peer %s -> ignored.", man.Title, p.UID[:8])
			continue
		}

		if man.FileSize == 0 && !n.Storage.HasNote(man.Hash) {
			if err := n.Storage.Save2db(man, models.Bucketlocal); err != nil {
				return err
			}
		}

		if man.FileSize > 0 && !n.Storage.FileExists(man.FileHash) && man.FileSize < mindlsize {
			err := n.DlFile(p, man.FileHash)

			if err != nil {
				log.Printf("[sync] failed to dl %s: %v -> ignored", man.FileHash[:8], err)
			}
			log.Printf("[sync] dl blob for note: %s -> ok", man.Title)
			if err := n.Storage.Save2db(man, models.Bucketlocal); err != nil {
				return err
			}
		}

		if err := n.Storage.Save2db(man, models.Bucketvirtual); err != nil {
			return err
		}
	}

	log.Printf("[sync] %s (%s) -> synced", p.Name, p.UID[:8])
	return nil
}
