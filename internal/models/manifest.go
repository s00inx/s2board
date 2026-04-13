// manifest for notes
package models

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type Manifest struct {
	// public info
	Title     string `json:"title"`
	Desc      string `json:"desc"`
	Timestamp int64  `json:"ts"`
	Version   int64  `json:"ver"`

	// file info (optional if note has file)
	FileHash string `json:"filehash,omitempty"`
	FileName string `json:"filename,omitempty"`
	FileSize int64  `json:"size"`

	// node info
	AuthorUID  string `json:"author"`
	AuthorName string `json:"author_name"`

	// manifest private info
	Hash      string `json:"hash"`
	Signature string `json:"sig"`
}

// make new manifest
func NewMan(title, desc, auuid, auname, fhash, fname string, fsize int64) *Manifest {
	return &Manifest{
		Title:      title,
		Desc:       desc,
		AuthorUID:  auuid,
		AuthorName: auname,
		FileHash:   fhash,
		FileName:   fname,
		FileSize:   fsize,
		Timestamp:  time.Now().Unix(),
		Version:    1,
	}
}

// NewMannofile — если лень передавать пустые строки для текста
func NewMannofile(title, desc, uid, name string) *Manifest {
	return NewMan(title, desc, uid, name, "", "", 0)
}

// struct for safe calculating hash
type hdata struct {
	Title     string `json:"title"`
	FileHash  string `json:"filehash,omitempty"`
	Desc      string `json:"desc"`
	Timestamp int64  `json:"ts"`
	AuthorUID string `json:"author"`
}

func (m *Manifest) buildhdata() *hdata {
	return &hdata{
		Title:     m.Title,
		FileHash:  m.FileHash,
		Desc:      m.Desc,
		Timestamp: m.Timestamp,
		AuthorUID: m.AuthorUID,
	}
}

// calculate sha-256 id for note
func (m *Manifest) CalcID() []byte {
	h := sha256.New()

	jb, _ := json.Marshal(m.buildhdata())
	h.Write(jb)
	return h.Sum(nil)
}

// sign manifest w note private key before pushing it to network
func (m *Manifest) Sign(privk ed25519.PrivateKey) error {
	hbytes := m.CalcID()
	m.Hash = hex.EncodeToString(hbytes)

	sig := ed25519.Sign(privk, hbytes)
	m.Signature = hex.EncodeToString(sig)

	return nil
}

// verify file hash
func (m *Manifest) Verify() bool {
	pubk, err := hex.DecodeString(m.AuthorUID)
	if err != nil || len(pubk) != ed25519.PublicKeySize {
		return false
	}
	sig, err := hex.DecodeString(m.Signature)
	if err != nil {
		return false
	}

	hbytes := m.CalcID()

	return ed25519.Verify(pubk, hbytes, sig)
}
