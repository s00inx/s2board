package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

func (s *Storage) fhash2dir(fhash string) string {
	shard := fhash[:2]
	return filepath.Join(s.Dir, "blobs", shard)
}

func (s *Storage) fhash2path(fhash string) string {
	shard := fhash[:2]
	return filepath.Join(s.Dir, "blobs", shard, fhash)
}

func (s *Storage) saveHashed(file *os.File, src string) (string, error) {
	hehash := sha256.New()
	if _, err := io.Copy(hehash, file); err != nil {
		return "", err
	}
	fhash := hex.EncodeToString(hehash.Sum(nil))
	tdir := s.fhash2dir(fhash)

	os.MkdirAll(tdir, 0755)
	tpath := filepath.Join(tdir, fhash)
	err := os.Link(src, tpath)
	if err != nil {
		// fallback
	}

	return fhash, nil
}

// регистрируем наш файл непосредственно открывая его на диске.
func (s *Storage) RegisterFile(src string) (string, int64, error) {
	file, err := os.Open(src)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	fhash, err := s.saveHashed(file, src)
	if err != nil {
		return "", 0, err
	}

	info, err := os.Stat(src)
	if err != nil {
		return "", 0, err
	}

	return fhash, info.Size(), nil
}

func (s *Storage) DeleteFile(fhash string) error {
	target := s.fhash2path(fhash)

	return os.Remove(target)
}
