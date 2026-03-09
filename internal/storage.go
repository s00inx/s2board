package internal

import (
	"encoding/json"
	"log"

	"go.etcd.io/bbolt"
)

// память это по сути прост обертка над бд
type Storage struct {
	db *bbolt.DB
}

// создаём объект
func InitStorage(path string) *Storage {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("entries"))
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	return &Storage{db: db}
}

// сохраняем энтри
func (s *Storage) SaveEntry(entry NoteMeta) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))

		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		return b.Put([]byte(entry.Hash), data)
	})
}

// проверка есть ли уже такой файл по хешу,
// нужен для синхронизации и обмена
func (s *Storage) HasEntry(hash string) bool {
	exists := false
	s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("entries"))
		v := b.Get([]byte(hash))
		if v != nil {
			exists = true
		}
		return nil
	})
	return exists
}
