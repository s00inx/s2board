// file with constants for clean code
package models

// bucket names for db (storage/)
const (
	Bucketlocal   = "local"
	Bucketvirtual = "virtual"
	Bucketfi      = "file_index"
)

// mdns service options (network/mdns)
const (
	ServiceName = "_s2board._tcp"
)

// p2ppacket action codes
type Actcode uint8

const (
	// files
	Actsave Actcode = iota + 1
	Actdel

	// sync & hello/bye
	ActHello
	ActHelloAck
	Actbye
	ActSync

	// exchange
	ActReqM
	ActRespM
	ActReqF
	ActRespF
)
