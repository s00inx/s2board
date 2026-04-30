package codec

import (
	"encoding/json"
	"io"
)

// RU: для MVP был выбран именно json, так как он человекочитаемый и наглядный + следую концепции REST
// для хорошего варианта json создаеи слишком большой оверхед

type JSONCodec struct{}

func (j JSONCodec) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j JSONCodec) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (j JSONCodec) EncodeStream(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func (j JSONCodec) DecodeStream(r io.ReadCloser, v any) error {
	return json.NewDecoder(r).Decode(v)
}
