// intenal database logic
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/s00inx/s2board/internal/models"
	"go.etcd.io/bbolt"
)

// RU: есть 2 бакета (таблицы) -
//     1: локальные файлы - те, которые лежат на диске (меняется ток при сохранении на диск, только внутри программы)
//     2: вся доска с манифестами - меняется при синхронизации и отдается на фронтенд

// save file to internal database
func (s *InternalStorage) Save2db(man models.Manifest, bucket string) error {
	if bucket != models.Bucketlocal && bucket != models.Bucketvirtual {
		return errors.New("[DB] invalid bucket")
	}

	return s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))

		data, err := json.Marshal(man)
		if err != nil {
			return err
		}

		err = b.Put([]byte(man.Hash), data)
		if err != nil {
			return err
		}

		if man.FileHash != "" {
			b = tx.Bucket([]byte(models.Bucketfi))
			return b.Put([]byte(man.FileHash), []byte(man.Hash))
		}

		return nil
	})
}

func (s *InternalStorage) GetManh(hash string, bucket string) (*models.Manifest, error) {
	var m models.Manifest

	err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
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

func (s *InternalStorage) GetManfh(fhash string, bucket string) (*models.Manifest, error) {
	var m models.Manifest

	err := s.DB.View(func(tx *bbolt.Tx) error {
		fb := tx.Bucket([]byte(models.Bucketfi))
		if fb == nil {
			return errors.New("[DB] no file index")
		}
		manhash := fb.Get([]byte(fhash))
		if manhash == nil {
			return errors.New("[DB] no file w this hash")
		}

		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return errors.New("bucket not found")
		}
		data := b.Get([]byte(manhash))
		if data == nil {
			return errors.New("not found")
		}

		return json.Unmarshal(data, &m)
	})

	return &m, err
}

func (s *InternalStorage) DeleteMan(hash string, bucket string) (string, error) {
	var fh string

	err := s.DB.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}

		data := b.Get([]byte(hash))
		if data == nil {
			return fmt.Errorf("manifest not found")
		}

		var m models.Manifest
		if err := json.Unmarshal(data, &m); err == nil {
			fh = m.FileHash
		}

		if err := b.Delete([]byte(hash)); err != nil {
			return err
		}

		// чистим и индекс
		if bucket == models.Bucketlocal {
			if i := tx.Bucket([]byte(models.Bucketfi)); i != nil {
				i.Delete([]byte(fh))
			}
		}

		return nil
	})

	return fh, err
}

// получить все хеши для синхронизации, берем из локальной бд.
// (так как это для синка, то есть интересуют тока те записи, которыми нода может поделиться)
func (s *InternalStorage) GetHashesList() ([]string, error) {
	var hashes []string

	err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(models.Bucketlocal))
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

func (s *InternalStorage) NoteExist(hash string) bool {
	var exists bool

	err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(models.Bucketlocal))
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

func (s *InternalStorage) GetManList() []models.Manifest {
	var manlist []models.Manifest

	err := s.DB.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(models.Bucketvirtual))
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

func (s *InternalStorage) Cleanvb() error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		err := tx.DeleteBucket([]byte(models.Bucketvirtual))
		if err != nil {
			return fmt.Errorf("[db] failed to delete virtual bucket: %v", err)
		}

		_, err = tx.CreateBucket([]byte(models.Bucketvirtual))
		if err != nil {
			return fmt.Errorf("[DB] failed to recreate virtual bucket: %v", err)
		}

		log.Println("[db] virtual bucket cleaned up")
		return nil
	})
}

func (s *InternalStorage) InitLocal() error {
	return s.DB.Update(func(tx *bbolt.Tx) error {
		local := tx.Bucket([]byte(models.Bucketlocal))
		if local == nil {
			return nil
		}

		virtual, err := tx.CreateBucketIfNotExists([]byte(models.Bucketvirtual))
		if err != nil {
			return err
		}

		return local.ForEach(func(k, v []byte) error {
			return virtual.Put(k, v)
		})
	})
}
