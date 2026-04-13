// манифест записи это по сути её метадата, здесь сама структура и её методы, конструктор и все, что касается хеша
package models

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"time"
)

// сам манифест записи
type Manifest struct {
	// приватные поля
	Hash      string `json:"hash"`
	Signature string `json:"sig"`
	FileHash  string `json:"filehash,omitempty"`

	// публичная информация
	Title   string `json:"title"`
	Content string `json:"content"`
	Desc    string `json:"desc"`

	AuthorUID  string `json:"author"`
	AuthorName string `json:"author_name"`

	Size      int64 `json:"size"`
	Timestamp int64 `json:"ts"` // unix-timestamp
}

// конструктор для новой заметки
func NewMan(title, content, desc, fileHash string, size int64) *Manifest {
	return &Manifest{
		Title:     title,
		Content:   content,
		Desc:      desc,
		FileHash:  fileHash,
		Size:      size,
		Timestamp: time.Now().Unix(),
	}
}

// структура для того чтобы считать хеш,
// мы берем толтко статические поля и считаем хеш от них в json
type hdata struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	FileHash  string `json:"filehash,omitempty"`
	Desc      string `json:"desc"`
	Timestamp int64  `json:"ts"`
	AuthorUID string `json:"author"`
}

func (m *Manifest) buildhdata() *hdata {
	return &hdata{
		Title:     m.Title,
		Content:   m.Content,
		FileHash:  m.FileHash,
		Desc:      m.Desc,
		Timestamp: m.Timestamp,
		AuthorUID: m.AuthorUID,
	}
}

// айди записи это 32 байта хеша статических полей
func (m *Manifest) CalcID() []byte {
	h := sha256.New()

	jb, _ := json.Marshal(m.buildhdata())

	io.Copy(h, bytes.NewReader(jb)) // считаем хеш от полного джсона
	return h.Sum(nil)
}

// подписать манифест перед тем как отправлять в сеть
func (m *Manifest) Sign(privk ed25519.PrivateKey) error {
	hbytes := m.CalcID()
	m.Hash = hex.EncodeToString(hbytes)

	sig := ed25519.Sign(privk, hbytes)
	m.Signature = hex.EncodeToString(sig)

	return nil
}

// проверить хкш
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
