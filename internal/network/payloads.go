// payload structs for p2p packets
package network

// for handhake/sync
type syncpl struct {
	Name   string   `json:"n"`
	UID    string   `json:"u"`
	Port   int      `json:"p"`
	Hashes []string `json:"h,omitempty"`
}

// file chunk request
type filereqpl struct {
	Fh     string `json:"fh"`
	Offset int64  `json:"o"`
	Size   int    `json:"s"`
}

// file bytes send
type fileresppl struct {
	Fh     string `json:"fh"`
	Offset int64  `json:"o"`
	Bytes  []byte `json:"b"`
}

// delete note
type delpl struct {
	Mhash string `json:"mh"`
}

// notify all peers about peer is new seed for file
type dlpl struct {
	Mhash string `json:"mh"`
}
