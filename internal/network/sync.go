// utils for sync node and peer
package network

import (
	"log"

	"github.com/s00inx/s2board/internal/models"
)

// async virtual board (frontend view) and internal database
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
			log.Printf("[sync] peer %s not found in active peers table!", pid[:8])
			continue
		}

		found, err := n.Fetch(p, hs)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, m := range found {
			if m.Verify() {
				err := n.InternalStorage.Save2db(m, models.Bucketvirtual)
				if err != nil {
					log.Printf("error saving a file")
					return
				}
				n.mpeers.add(m.Hash, p.UID)
				log.Println("m verified", m)
			}
		}
	}
}

// get remaining hashes for sync from mhtable
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
func (n *Node) getsynclist() ([]string, error) {
	return n.InternalStorage.GetHashesList(models.Bucketlocal)
}

// forget about peer when it leave the network
func (n *Node) forgetpeer(puid string) {
	log.Printf("[sync] peer %s left -> syncing", puid[:8])
	oh := n.mpeers.droppeer(puid)
	if len(oh) > 0 {
		log.Printf("[sync] %d hashes unavailable after %s leave -> removing...", len(oh), puid[:8])
	}

	n.peers.rm(puid)
	go n.syncbye(oh)
}

// async virtual board and orphaned hashes after peer leave
func (n *Node) syncbye(hl []string) {
	var rmh int
	for _, hash := range hl {
		rmpeers, _ := n.mpeers.getpeerlist(hash)

		if len(rmpeers) == 0 && !n.InternalStorage.NoteExist(hash) {
			rmh++
			n.InternalStorage.DeleteMan(hash, models.Bucketvirtual)
		}
	}

	log.Printf("[sync] removed %d/%d hashes", rmh, len(hl))
}
