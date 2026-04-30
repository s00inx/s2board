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

type mpeertable struct {
	mu sync.RWMutex
	d  map[string][]string
}

func newmpeertable() *mpeertable {
	return &mpeertable{
		d: make(map[string][]string),
	}
}

func (mt *mpeertable) add(mhash string, peerid string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if slices.Contains(mt.d[mhash], peerid) {
		return
	}
	mt.d[mhash] = append(mt.d[mhash], peerid)
}

func (mt *mpeertable) getpeerlist(mhash string) ([]string, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	h, ok := mt.d[mhash]
	return h, ok
}

func (mt *mpeertable) rm(peerid string) []string {
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

func (mt *mpeertable) drop(mhash string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	delete(mt.d, mhash)
}

// file hash -> peers table for downloading
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
