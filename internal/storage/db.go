// все что касается внутренней бд
// !!: здесь може быть любая бд, выбран бболт из-за скорости и безопасности данных (но можно использовать и sql, все что угодно)
// см. network/node -> nodeStorage
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/s00inx/s2board/internal/models"
	"go.etcd.io/bbolt"
)

// есть 2 бакета (таблицы) -
// 1: локальные файлы - те, которые лежат на диске (меняется ток при сохранении на диск, только внутри программы)
// 2: вся доска с манифестами - меняется при синхронизации и отдается на фронтенд

// сохранить файл в бд в:
// local (когда сохраняем 100% на диске)
// virtual (все манифесты сюда)
func (s *Storage) Save2db(man models.Manifest, bucket string) error {
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

// взять манифест из бд по хешу
func (s *Storage) Getmanh(hash string, bucket string) (*models.Manifest, error) {
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

// взять манифест из бд по хешу файла (для этого пользуемся доп. индексами)
func (s *Storage) Getmanfh(fhash string, bucket string) (*models.Manifest, error) {
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

// удалить запись из бд по хешу (и вернуть хеш файла)
func (s *Storage) Delman(hash string, bucket string) (string, error) {
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
func (s *Storage) GetHashesList() ([]string, error) {
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

// есть ли в бакете такая запись?
// это нужно для скачивания -> есть смылсл проверять только в локале
func (s *Storage) HasNote(hash string) bool {
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

// получить все манифесты для фронтенда (то есть ищем только в виртуал)
func (s *Storage) GetManlist() []models.Manifest {
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

// очистить виртуальную доску
func (s *Storage) CleanVirtual() error {
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

func (s *Storage) RepubLocal() error {
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
