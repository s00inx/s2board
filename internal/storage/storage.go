package storage

import (
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

const (
	dbname = "s2.db"
)

// Storage должен удовлетворять интерфейсу nodeStorage из network
type Storage struct {
	DB  *bbolt.DB // ссылка на экземпляр бд
	Dir string    // директррия с файлами и ключами (../data по дефолту)
}

// инициализировать новый экземпляр стораджа, нужно вызвать 1 раз при инициализации сервиса,
// в качестве параметра указываем базовую директорию
func Init(dir string) (*Storage, error) {
	os.MkdirAll(dir, 0755)

	db, err := bbolt.Open(filepath.Join(dir, dbname), 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("entries"))
		return err
	})

	return &Storage{Dir: dir, DB: db}, nil
}
