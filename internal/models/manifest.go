package models

import "crypto/ed25519"

/*
отправить манифест POST /api/share -> json с манифестом
*/

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
	FileHash string `json:"filehash.omitempty"`
	Desc     string `json:"desc"`
	Size     int64  `json:"size"`

	// для фронтенда)) здесь цвет там и все вот это че надо
	Type string `json:"type"`
}

func (m *NoteManifest) Publish() {

}

func (m *NoteManifest) SignManifest(pk ed25519.PrivateKey) {

}
