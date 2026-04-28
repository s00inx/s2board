package network

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/s00inx/s2board/internal/models"
)

const (
	mindlsize = 1
	// mindlsize = 25 * 1 << 20
)

// INTERNAL storage interface.
// all interactions with interanl db (bbolt, sql etc.)
type nodeInternalStorage interface {
	Save2db(man models.Manifest, bucket string) error
	GetManList() []models.Manifest
	GetHashesList() ([]string, error)
	GetManh(hash string, bucket string) (*models.Manifest, error)
	GetManfh(fhash string, bucket string) (*models.Manifest, error)
	NoteExist(hash string) bool
	Cleanvb() error
	DeleteMan(hash string, bucket string) (string, error)
	InitLocal() error
}

// node external storage interface
// all interaction with physical disk
type nodeExternalStorage interface {
	Save2disk(src string) (string, int64, error)
	FileExists(fhash string) bool
	SaveFile(fhash string, r io.Reader) error
	Fhash2path(fhash string) string
	DeleteFile(fhash string) error
}

// upload file TO local network
func (n *Node) Uploadf(src, title, desc string) (*models.Manifest, error) {
	var fhash string
	var fsize int64
	var err error

	if src != "" {
		fhash, fsize, err = n.FileStorage.Save2disk(src)
		if err != nil {
			return nil, fmt.Errorf("disk err: %w", err)
		}
	}

	man := models.NewMan(
		title,
		desc,
		n.UID,
		n.PubName,
		fhash,
		filepath.Base(src),
		fsize,
	)

	if err := man.Sign(n.PrivateK); err != nil {
		return nil, fmt.Errorf("failed to sign manifest: %w", err)
	}

	man.Hash = hex.EncodeToString(man.CalcID())

	if err = n.DbStorage.Save2db(*man, models.Bucketvirtual); err != nil {
		log.Printf("failed to sync self to virtual: %v", err)
	}

	// save to db bc we saved it to disk
	if err = n.DbStorage.Save2db(*man, models.Bucketlocal); err != nil {
		return nil, fmt.Errorf("db err: %w", err)
	}
	return man, nil
}

// receive file from net -> process
func (n *Node) Recvf(p Peer, man *models.Manifest) error {
	n.filepeers.add(man.FileHash, p)

	if err := n.DbStorage.Save2db(*man, models.Bucketvirtual); err != nil {
		log.Printf("[db] failed to save to virtual: %v", err)
	}

	// 3. file was flushed to disk -> save that to local
	if man.FileSize > 0 && n.FileStorage.FileExists(man.FileHash) {
		return n.DbStorage.Save2db(*man, models.Bucketlocal)
	}

	// 4. auto-download blobs under max size limit
	if man.FileSize <= mindlsize && man.FileSize > 0 {
		go func() {
			err := n.DlFile(p, man.FileHash)
			if err == nil {
				n.DbStorage.Save2db(*man, models.Bucketlocal)
				log.Printf("[download] success: %s", man.FileHash[:8])
			}
		}()
	}

	// 5. ask peers for file availability -> add them to fpeermap
	if man.FileSize > 0 {
		go n.askpeers(man.FileHash)
	}

	log.Printf("[recv] %s from %s:%d", man.Hash[:8], p.IP, p.Port)

	return nil
}

// ask all available peers about file
func (n *Node) askpeers(fhash string) {
	n.peers.mu.Lock()
	peers := n.peers.d
	n.peers.mu.Unlock()

	for _, peer := range peers {
		go func(p Peer) {
			url := fmt.Sprintf("http://%s:%d/api/hasf/%s", p.IP, p.Port, fhash)
			resp, err := n.client.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				n.filepeers.add(fhash, p)
				log.Printf("[found] file %s also available on %s", fhash[:8], p.IP)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}(peer)
	}
}

// download file from peer using only file hash
func (n *Node) Dlf(fhash string) (io.ReadCloser, error) {
	if n.FileStorage.FileExists(fhash) {
		return os.Open(n.FileStorage.Fhash2path(fhash))
	}

	hostl, ok := n.filepeers.getpeerlist(fhash)

	fmt.Println(hostl)
	if !ok || len(hostl) == 0 {
		return nil, fmt.Errorf("no peers found for hash %s", fhash[:8])
	}

	for _, fpeerid := range hostl {
		fpeer, ok := n.peers.d[fpeerid]
		if !ok {
			continue
		}

		dsturlp := fmt.Sprintf("http://%s:%d/api/dl/%s", fpeer.IP, fpeer.Port, fhash)

		resp, err := n.client.Get(dsturlp)
		if err != nil {
			log.Printf("[proxy] peer %s error: %v", fpeerid[:8], err)
			n.ForgetPeer(fpeerid)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}

		resp.Body.Close()
	}

	return nil, fmt.Errorf("failed to retrieve file from %d peers", len(hostl))
}

func (n *Node) DlFile(p Peer, fhash string) error {
	dsturl := fmt.Sprintf("http://%s:%d/api/dl/%s", p.IP, p.Port, fhash)

	resp, err := n.client.Get(dsturl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[dl file] peer returned status: %d", resp.StatusCode)
	}

	h := sha256.New()
	tr := io.TeeReader(resp.Body, h)

	if err := n.FileStorage.SaveFile(fhash, tr); err != nil {
		return err
	}

	realhash := hex.EncodeToString(h.Sum(nil))
	if fhash != realhash {
		n.FileStorage.DeleteFile(fhash)
		return fmt.Errorf("[dl file] hash mismatch: expected %s, got %s", fhash[:8], realhash[:8])
	}

	return nil
}

func (n *Node) RmNote(hash string) error {
	fhash, err := n.DbStorage.DeleteMan(hash, models.Bucketlocal)
	if err != nil {
		return fmt.Errorf("")
	}

	if _, err := n.DbStorage.DeleteMan(hash, models.Bucketvirtual); err != nil {
		return fmt.Errorf("")
	}

	if fhash != "" {
		return n.FileStorage.DeleteFile(fhash)
	}

	return nil
}
