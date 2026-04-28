package network

import (
	"slices"
	"sync"
	"time"
)

// peer is abstraction above a certain device in local network
type Peer struct {
	Name     string
	UID      string
	IP       string
	Port     int
	LastSeen time.Time
}

func NewPeer(uid string) *Peer {
	return &Peer{UID: uid}
}

// file -> peers table for downloading
type fpeermap struct {
	mu sync.Mutex
	d  map[string][]string
}

func newfilepeermap() *fpeermap {
	return &fpeermap{
		d: make(map[string][]string, 0),
	}
}

func (fm *fpeermap) add(hash string, peer Peer) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if slices.Contains(fm.d[hash], peer.UID) {
		return
	}
	fm.d[hash] = append(fm.d[hash], peer.UID)
}

func (fm *fpeermap) rm(peerid string) []string {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	var emptyh []string

	for hash, ps := range fm.d {
		nps := ps[:0]
		for _, pr := range ps {
			if pr != peerid {
				nps = append(nps, pr)
			}
		}
		fm.d[hash] = nps

		if len(nps) == 0 {
			emptyh = append(emptyh, hash)
			delete(fm.d, hash)
		}
	}
	return emptyh
}

func (fm *fpeermap) getpeerlist(hash string) ([]string, bool) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	h, ok := fm.d[hash]
	return h, ok
}

// uid -> peer map for tracking active connections in local network
type peermap struct {
	d  map[string]Peer
	mu sync.Mutex
}

func newpeermap() *peermap {
	return &peermap{
		d: make(map[string]Peer, 0),
	}
}

func (pm *peermap) add(p Peer) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.d[p.UID] = p
}

func (pm *peermap) rm(uid string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.d, uid)
}

func (pm *peermap) getpeer(uid string) (Peer, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	p, ok := pm.d[uid]
	return p, ok
}
