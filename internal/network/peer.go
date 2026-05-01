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

// manifest hash -> peer list table
// manifest guarantee that file exist on NODE disk, because handshake gives only "local" files
type mhtable struct {
	mu sync.RWMutex
	d  map[string][]string
}

func newmpeertable() *mhtable {
	return &mhtable{
		d: make(map[string][]string),
	}
}

func (mt *mhtable) add(mhash string, peerid string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if slices.Contains(mt.d[mhash], peerid) {
		return
	}
	mt.d[mhash] = append(mt.d[mhash], peerid)
}

func (mt *mhtable) getpeerlist(mhash string) ([]string, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	h, ok := mt.d[mhash]
	return h, ok
}

func (mt *mhtable) droppeer(peerid string) []string {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	var emptyh []string

	for mhash, ps := range mt.d {
		nps := ps[:0]
		for _, pr := range ps {
			if pr != peerid {
				nps = append(nps, pr)
			}
		}
		mt.d[mhash] = nps

		if len(nps) == 0 {
			emptyh = append(emptyh, mhash)
			delete(mt.d, mhash)
		}
	}
	return emptyh
}

func (mt *mhtable) dropfh(mhash string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	delete(mt.d, mhash)
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
