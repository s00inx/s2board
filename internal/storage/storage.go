package storage

import (
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// Storage должен удовлетворять интерфейсу nodeStorage из network
type Storage struct {
	DB  *bbolt.DB // ссылка на экземпляр бд
	Dir string    // директррия с файлами и ключами (../data по дефолту)
}

func NewStorage(dir string) (*Storage, error) {
	os.MkdirAll(dir, 0755)

	db, err := bbolt.Open(filepath.Join(dir, "s2.db"), 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("entries"))
		return err
	})

	return &Storage{Dir: dir, DB: db}, nil
}
