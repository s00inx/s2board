// utils for sync node and peer
package network

import (
	"fmt"
	"log"

	"github.com/s00inx/s2board/internal/models"
)

// async virtual board (frontend view) and internal database
func (n *Node) synchello() {
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

		found, err := n.fetchm(p, hs)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, m := range found {
			if m.Verify() {
				n.InternalStorage.Save2db(m, models.Bucketvirtual)
				if m.FileSize == 0 {
					n.InternalStorage.Save2db(m, models.Bucketlocal)
					n.mpeers.add(m.Hash, n.UID)
				}
				n.mpeers.add(m.Hash, p.UID)
				log.Printf("[sync] manifest %s indexed for peer %s", m.Hash[:8], p.UID[:8])
			} else {
				log.Printf("[ERR] manifest %s verification failed!", m.Hash[:8])
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

		if len(rmpeers) == 0 {
			m, _ := n.InternalStorage.GetManh(hash, models.Bucketvirtual)

			if m != nil && n.FileStorage.FileExists(m.FileHash) {
				n.mpeers.add(hash, n.UID)
				log.Printf("[sync] i am the new source for %s", hash[:8])
				continue
			}

			rmh++
			n.InternalStorage.DeleteMan(hash, models.Bucketvirtual)
		}
	}
	log.Printf("[sync] removed %d/%d hashes", rmh, len(hl))
}

// fetch a batch of manifests from peer
func (n *Node) fetchm(p Peer, h []string) ([]models.Manifest, error) {
	log.Printf("[fetch] %d from %s (%s)", len(h), p.UID[:8], p.Name)

	data, err := n.Codec.Encode(h)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[sync] hashlist encode error")
	}

	reqp := models.NewPacket(data, models.ActReqM, n.UID, n.PrivateK)
	inc, err := n.Transport.Sendp(p.IP, p.Port, reqp)
	if err != nil {
		log.Println("transport incp error: ", err)
		return nil, err
	}

	var pl []models.Manifest
	if err := n.Codec.Decode(inc.Payload, &pl); err != nil {
		return nil, err
	}

	log.Printf("[fetch] -> %d/%d from %s", len(pl), len(h), inc.Senderuid[:8])

	return pl, nil
}
