package storage

import (
	"os"
	"path/filepath"

	"github.com/s00inx/s2board/internal/models"
	"go.etcd.io/bbolt"
)

const (
	dbname = "s2.db"
)

type InternalStorage struct {
	DB *bbolt.DB
}

type ExternalStorage struct {
	Dir      string
	BlobsDir string
}

// инициализировать новый экземпляр стораджа, нужно вызвать 1 раз при инициализации сервиса,
// в качестве параметра указываем базовую директорию
func Init(dir string) (*InternalStorage, *ExternalStorage, error) {
	os.MkdirAll(dir, 0755)

	db, err := bbolt.Open(filepath.Join(dir, dbname), 0600, nil)
	if err != nil {
		return nil, nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(models.Bucketlocal))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(models.Bucketvirtual))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(models.Bucketfi))
		return err
	})

	return &InternalStorage{DB: db}, &ExternalStorage{Dir: dir}, nil
}
