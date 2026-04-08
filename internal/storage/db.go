// все что касается внутренней бд
// !!: здесь може быть любая бд, выбран бболт из-за скорости и безопасности данных (но можно использовать и sql, все что угодно)
// см. network/node -> nodeStorage
// в бд хранятся ТОЛЬКО файлы, которые есть непосредственно на диске, то есть которые узел может раздать
package storage

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/s00inx/s2board/internal/models"
	"go.etcd.io/bbolt"
)

// сохранить файл в бд
func (s *Storage) Save2db(man models.Manifest) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))

		data, err := json.Marshal(man)
		if err != nil {
			return err
		}

		return b.Put([]byte(man.Hash), data)
	})
}

// взять манифест из бд по хешу
func (s *Storage) GetFMan(hash string) (*models.Manifest, error) {
	var m models.Manifest

	err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		data := b.Get([]byte(hash))
		if data == nil {
			return fmt.Errorf("not found")
		}

		return json.Unmarshal(data, &m)
	})

	return &m, err
}

// удалить запись из бд по хешу
func (s *Storage) DelManifest(hash string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		var m models.Manifest
		data := b.Get([]byte(hash))
		if data != nil {
			json.Unmarshal(data, &m)
			s.DeleteFile(m.FileHash)
		}

		return b.Delete([]byte(hash))
	})
}

// получить все хеши для синхронизации
func (s *Storage) GetHashesList() ([]string, error) {
	var hashes []string

	err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			hashes = append(hashes, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return hashes, nil
}

func (s *Storage) HasNote(hash string) bool {
	var exists bool

	err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("manifests"))
		if b == nil {
			return nil
		}

		if v := b.Get([]byte(hash)); v != nil {
			exists = true
		}
		return nil
	})

	if err != nil {
		log.Printf("[ERR] storage hasnote: %v", err)
		return false
	}

	return exists
}

func (s *Storage) GetManlist() []models.Manifest {
	var manlist []models.Manifest

	err := s.DB.View(func(tx *bbolt.Tx) error {
		// Выбираем бакет с манифестами
		b := tx.Bucket([]byte("manifests"))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var m models.Manifest

			if err := json.Unmarshal(v, &m); err != nil {
				log.Printf("[WARN] failed to unmarshal manifest %s: %v", string(k), err)
				return nil
			}

			manlist = append(manlist, m)
			return nil
		})
	})

	if err != nil {
		log.Printf("[ERR] storage getmans: %v", err)
	}

	return manlist
}
