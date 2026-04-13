package storage

import (
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

const (
	dbname = "s2.db"

	localbucket = "local"
	virtbucket  = "virtual"
	fibucket    = "file_index"
)

// Storage должен удовлетворять интерфейсу nodeStorage из network
type Storage struct {
	DB  *bbolt.DB // ссылка на экземпляр бд
	Dir string    // директория с файлами и ключами (../data по дефолту)
}

// инициализировать новый экземпляр стораджа, нужно вызвать 1 раз при инициализации сервиса,
// в качестве параметра указываем базовую директорию
func Init(dir string) (*Storage, error) {
	os.MkdirAll(dir, 0755)

	db, err := bbolt.Open(filepath.Join(dir, dbname), 0600, nil)
	if err != nil {
		return nil, err
	}

	// создаем бакет для манифестов файлов которые уже есть на диске (хеш манифеста : манифест)
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(localbucket))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte("file_index"))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(virtbucket))
		return err
	})

	return &Storage{Dir: dir, DB: db}, nil
}
