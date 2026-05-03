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
	ActCreate Actcode = iota + 1
	ActDelete

	// sync & hello/bye
	ActHelloSyn
	ActRespHello
	ActBye
	ActSync
	ActDl

	// exchange
	ActReqM
	ActRespM
	ActReqF
	ActRespF
)
