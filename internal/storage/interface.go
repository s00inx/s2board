package storage

import (
	"io"

	"github.com/s00inx/s2board/internal/models"
)

// INTERNAL storage interface.
// all interactions with internal db (bbolt, sql etc.)
type NodeInternalStorage interface {
	Save2db(man models.Manifest, bucket string) error
	GetManList() []models.Manifest
	GetHashesList(bucket string) ([]string, error)
	GetManh(hash string, bucket string) (*models.Manifest, error)
	GetManfh(fhash string, bucket string) (*models.Manifest, error)
	NoteExist(hash string) bool
	Cleanvb() error
	DeleteMan(hash string, bucket string) error
	InitLocal() error
}

// external storage interface
// all interaction with physical disk
type NodeExternalStorage interface {
	Save2disk(src string) (string, int64, error)
	FileExists(fhash string) bool
	SaveFile(fhash string, r io.Reader) error
	Fhash2path(fhash string) string
	DeleteFile(fhash string) error
}
