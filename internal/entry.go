package internal

// метаданные записи на доске, которые хранятся и передаются
type NoteMeta struct {
	Hash      string `json:"hash"`
	Content   string `json:"text"`
	Timestamp int64  `json:"ts"`
	Type      string `json:"type"`
}
