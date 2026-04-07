package models

import "time"

type Peer struct {
	Name string
	UID  string
	IP   string
	Port int

	LastSeen time.Time
}
