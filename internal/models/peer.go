package models

import "time"

type Peer struct {
	UID  string
	IP   string
	Port int

	LastSeen time.Time
}
