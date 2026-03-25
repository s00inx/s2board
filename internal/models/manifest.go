// манифест записи это по сути её метадата, здесь сама структура и её методы, конструктор и все, что касается хеша
package models

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"time"
)

// сам манифест записи
type NoteManifest struct {
	// приватные поля
	Hash      string `json:"hash"`
	AuthorUID string `json:"author"`
	Signature string `json:"sig"`
	Timestamp int64  `json:"ts"` // unix-timestamp ofc

	// публичная информация
	Title    string `json:"title"`
	Content  string `json:"content"`
	FileHash string `json:"filehash,omitempty"`
	Desc     string `json:"desc"`
	Size     int64  `json:"size"`

	// для фронтенда)) здесь цвет там и все вот это че надо
	Type string `json:"type"`
}

func NewNote(title, content, desc, fileHash string, size int64, nType string) *NoteManifest {
	return &NoteManifest{
		Title:    title,
		Content:  content,
		Desc:     desc,
		FileHash: fileHash,
		Size:     size,
		Type:     nType,

		Timestamp: time.Now().Unix(),
	}
}

// айди записи это 32 байта хеша всех значимых полей.
func (m *NoteManifest) CalculateID() []byte {
	h := sha256.New()
	h.Write([]byte(m.AuthorUID))

	h.Write([]byte(m.Title))
	h.Write([]byte(m.Desc))

	tsbuf := make([]byte, 8)
	binary.BigEndian.PutUint64(tsbuf, uint64(m.Timestamp))
	h.Write(tsbuf)

	h.Write([]byte(m.Content))
	h.Write([]byte(m.FileHash))

	sizeBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBuf, uint64(m.Size))
	h.Write(sizeBuf)

	h.Write([]byte(m.Type))

	return h.Sum(nil)
}

// подписать манифест перед тем как отправлять в сеть
func (m *NoteManifest) Sign(privKey ed25519.PrivateKey) error {
	hbytes := m.CalculateID()

	m.Hash = hex.EncodeToString(hbytes)

	sig := ed25519.Sign(privKey, hbytes)
	m.Signature = hex.EncodeToString(sig)

	return nil
}

// проверить хкш
func (m *NoteManifest) Verify() bool {
	pubk, _ := hex.DecodeString(m.AuthorUID)
	sig, _ := hex.DecodeString(m.Signature)

	hbytes := m.CalculateID()

	return ed25519.Verify(pubk, hbytes, sig)
}
