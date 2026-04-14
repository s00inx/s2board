// program local content-adressed storage methods
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// file hash -> dir with file
func (s *Storage) Fhash2dir(fhash string) string {
	shard := fhash[:2]
	return filepath.Join(s.Dir, s.BlobDirName, shard)
}

// file hash -> path to file
func (s *Storage) Fhash2path(fhash string) string {
	shard := fhash[:2]
	return filepath.Join(s.Dir, s.BlobDirName, shard, fhash)
}

// save file with his hash (hard link or copy)
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

	// error here only if files and exec file on different disks
	// so we have fallback - file will copy to this dir
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

// save file to disk
func (s *Storage) Save2disk(src string) (string, int64, error) {
	file, err := os.Open(src)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	fhash, err := s.savetopath(file, src)
	if err != nil {
		return "", 0, err
	}

	info, err := os.Stat(src)
	if err != nil {
		return "", 0, err
	}
	return fhash, info.Size(), nil
}

// download local file
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

// delete file from local storage
func (s *Storage) Delfile(fhash string) error {
	target := s.Fhash2path(fhash)

	if err := os.Remove(target); err != nil {
		return err
	}

	pdir := filepath.Dir(target)
	os.Remove(pdir)

	return nil
}

// check if file exist in local storage
func (s *Storage) FileExists(fhash string) bool {
	_, err := os.Stat(s.Fhash2path(fhash))
	return err == nil
}
