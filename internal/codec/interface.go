package codec

import "io"

// RU: отдельный интерфейс, чтобы можно было менять логику шифрования данных без изменения логики программы

type Codec interface {
	// encode struct -> bytes
	Encode(v any) ([]byte, error)
	// bytes -> struct
	Decode(data []byte, v any) error
	// bytes stream -> struct
	DecodeStream(r io.ReadCloser, v any) error
	// struct -> steram writer
	EncodeStream(w io.Writer, v any) error
}
