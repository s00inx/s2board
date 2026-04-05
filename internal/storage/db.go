package storage

import (
	"encoding/json"
	"fmt"

	"github.com/s00inx/stdesk/internal/models"
	"go.etcd.io/bbolt"
)

func (s *Storage) SaveFile(man models.NoteManifest) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))

		data, err := json.Marshal(man)
		if err != nil {
			return err
		}

		return b.Put([]byte(man.Hash), data)
	})
}

func (s *Storage) GetManifest(hash string) (*models.NoteManifest, error) {
	var m models.NoteManifest

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

func (s *Storage) DeleteManifest(hash string) error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		var m models.NoteManifest
		data := b.Get([]byte(hash))
		if data != nil {
			json.Unmarshal(data, &m)
			s.DeleteFile(m.FileHash)
		}

		return b.Delete([]byte(hash))
	})
}
