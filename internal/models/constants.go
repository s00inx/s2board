// filw with constants for clean code
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

type Actcode uint8

const (
	Actsave Actcode = iota + 1
	Actdel
)
