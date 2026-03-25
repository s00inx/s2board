package storage

import "go.etcd.io/bbolt"

// Storage должен удовлетворять интерфейсу nodeStorage из network
type Storage struct {
	DB *bbolt.DB

	BaseDir string
	KeyDir  string
}
