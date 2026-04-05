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

// сохранить файл по хешу (либо сделать хардлинк, либо скопировать полностью)
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

	// ошибка тут если бинарник и файл на разных дисках, все равно придется копировать фвйл на диск с бинарником
	// если они на одном диске, сработает хардлинк
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

// удалить файл из памяти (точнее удалить линк, сам файл из памяти никуда не денется)
func (s *Storage) DeleteFile(fhash string) error {
	target := s.fhash2path(fhash)

	return os.Remove(target)
}

// FileExists проверяет наличие файла по его хешу
func (s *Storage) FileExists(fhash string) bool {
	_, err := os.Stat(s.fhash2path(fhash))
	return err == nil
}

// SaveBlob принимает Reader (тело ответа от другой ноды) и пишет в файл
func (s *Storage) SaveBlob(fhash string, r io.Reader) error {
	tpath := s.fhash2path(fhash)
	os.MkdirAll(filepath.Dir(tpath), 0755)

	f, err := os.Create(tpath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}
