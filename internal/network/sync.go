// sync node and peer
package network

import (
	"log"

	"github.com/s00inx/s2board/internal/models"
)

// get remaining hashes and sync virtual board
func (n *Node) syncvirtual() {
	m := n.getrhashes()
	if len(m) == 0 {
		log.Printf("[sync] nothing to sync -> skipped")
		return
	}

	tasks := make(map[string][]string)
	for _, mh := range m {
		peers, _ := n.mpeers.getpeerlist(mh)
		if len(peers) > 0 {
			pid := peers[0]
			tasks[pid] = append(tasks[pid], mh)
		}
	}

	for pid, hs := range tasks {
		p, exists := n.peers.getpeer(pid)
		if !exists || p.IP == "" || p.Port == 0 {
			return
		}

		found, err := n.Fetch(p, hs)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, m := range found {
			if m.Verify() {
				n.InternalStorage.Save2db(m, models.Bucketvirtual)
				n.mpeers.add(m.Hash, p.UID)
			}
		}
	}
}

func (n *Node) getrhashes() []string {
	var t []string
	n.mpeers.mu.RLock()

	for h := range n.mpeers.d {
		if !n.InternalStorage.NoteExist(h) {
			t = append(t, h)
		}
	}
	n.mpeers.mu.RUnlock()
	return t
}

// get list if all hashes
func (n *Node) getSyncList() ([]string, error) {
	return n.InternalStorage.GetHashesList()
}

// forget about peer when it leave the network
func (n *Node) forgetpeer(puid string) {
	log.Printf("[sync] peer %s left -> syncing", puid[:8])
	oh := n.mpeers.droppeer(puid)
	if len(oh) > 0 {
		log.Printf("%d hashes unavailable after %s leave", len(oh), puid[:8])
	}

	n.peers.rm(puid)
	n.syncvirtual()
}
