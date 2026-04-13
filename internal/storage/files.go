package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

const (
	blobdirname = "blobs"
)

// file hash -> dir with file
func (s *Storage) Fhash2dir(fhash string) string {
	shard := fhash[:2]
	return filepath.Join(s.Dir, blobdirname, shard)
}

// file hash -> path to file
func (s *Storage) Fhash2path(fhash string) string {
	shard := fhash[:2]
	return filepath.Join(s.Dir, blobdirname, shard, fhash)
}

// сохранить файл по хешу (либо сделать хардлинк, либо скопировать полностью)
func (s *Storage) savetopath(file *os.File, src string) (string, error) {
	hehash := sha256.New()
	if _, err := io.Copy(hehash, file); err != nil {
		return "", err
	}

	fhash := hex.EncodeToString(hehash.Sum(nil))
	tdir := s.Fhash2dir(fhash)

	os.MkdirAll(tdir, 0755)
	tpath := filepath.Join(tdir, fhash)
	err := os.Link(src, tpath)

	// ошибка тут если бинарник и файл на разных дисках, все равно придется копировать фвйл на диск с бинарником
	// если они на одном диске, сработает хардлинк
	if err != nil {
		dstf, err := os.Create(tpath)
		if err != nil {
			return "", err
		}
		defer dstf.Close()
		file.Seek(0, io.SeekStart)

		_, err = io.Copy(dstf, file)
		return fhash, err
	}

	return fhash, nil
}

// регистрируем наш файл непосредственно открывая его на диске.
func (s *Storage) Save2disk(src string) (string, int64, error) {
	// открываем файл
	file, err := os.Open(src)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	// сохраняем физически на диск
	fhash, err := s.savetopath(file, src)
	if err != nil {
		return "", 0, err
	}

	// узнаем инфо о нем, чтобы отправить данные для сохранения
	info, err := os.Stat(src)
	if err != nil {
		return "", 0, err
	}
	return fhash, info.Size(), nil
}

// сохранить скачанный файл на диск
func (s *Storage) SaveBlob(fhash string, r io.Reader) error {
	tpath := s.Fhash2path(fhash)
	os.MkdirAll(filepath.Dir(tpath), 0755)

	f, err := os.Create(tpath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

// удалить файл из локальной памяти программы
func (s *Storage) Delfile(fhash string) error {
	target := s.Fhash2path(fhash)

	if err := os.Remove(target); err != nil {
		return err
	}

	// если директория не пустая, она не удалится
	pdir := filepath.Dir(target)
	os.Remove(pdir)

	return nil
}

// проверить наличие файла в локальном хранилище программы по его хешу
func (s *Storage) FileExists(fhash string) bool {
	_, err := os.Stat(s.Fhash2path(fhash))
	return err == nil
}
