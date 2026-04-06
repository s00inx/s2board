// синхронизация конкретной ноды с другими в этой же локальной сети
// получить ВСЕ хеши -> сравнить -> запросить манифесты -> проверить -> сохранить
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

// получить список всех хешей которые есть у конкретной ноды
func (n *Node) GetHashes() ([]string, error) {
	return n.Storage.GetHashes()
}

// принимает список хешей и отдает полные манифесты
func (n *Node) FetchManifests(hashes []string) ([]models.NoteManifest, error) {
	var res []models.NoteManifest

	for _, h := range hashes {
		man, err := n.Storage.GetManifest(h)
		if err == nil && man != nil {
			res = append(res, *man)
		}
	}

	return res, nil
}

// полностью синхронизировать 2 ноды между собой в несколько этапов
func (n *Node) Syncw(p models.Peer) error {
	// делаю одного клиента с таймаутом
	c := &http.Client{Timeout: 10 * time.Second}

	// 1: спрашиваю у ноды список хешей всех записей которые у нее есть
	resp, err := c.Get(fmt.Sprintf("http://%s:%d/api/sync", p.IP, p.Port))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var excl []string
	if err := json.NewDecoder(resp.Body).Decode(&excl); err != nil {
		return err
	}
	var missing []string // собираю список нужных хешей
	for _, h := range excl {
		_, err := n.Storage.GetManifest(h)
		if err != nil {
			missing = append(missing, h)
		}
	}
	// нечего качать, завершаем
	if len(missing) == 0 {
		return nil
	}

	log.Printf("[SYNC] found %d missing notes from %s, fetching...", len(missing), p.UID[:8])

	// 2: делаем запрос к ноде, чтобы она выдала нам манифеств только по нужным хешам
	body, _ := json.Marshal(missing)
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

	// добавляем манифесты в свое локальное хранилище
	for _, man := range newnotes {
		// проверяем ноду
		if !man.Verify() {
			log.Printf("[WARNING] fake signature for note %s from peer %s --> ignored.", man.Hash, p.UID[:8])
			continue
		}

		// if man.FileHash != "" && !n.Storage.FileExists(man.FileHash) {
		// 	log.Printf("[SYNC] downloading blob for note: %s", man.Title)
		// 	err := n.DlBlob(p, man.FileHash)
		// 	if err != nil {
		// 		log.Printf("[ERR] failed to download blob %s: %v", man.FileHash[:8], err)
		// 	}
		// }

		err = n.Storage.SaveFile(man)
		if err == nil {
			log.Printf("[SYNC] added new note: %s", man.Title)
		}
	}

	return nil
}
