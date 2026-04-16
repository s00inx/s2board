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
	"slices"
	"sync"

	"github.com/s00inx/s2board/internal/models"
)

const (
	mindlsize = 1
	// mindlsize = 25 * 1 << 20
)

type fpeermap struct {
	mu sync.Mutex
	d  map[string][]string
}

func newfilepeermap() *fpeermap {
	return &fpeermap{
		d: make(map[string][]string, 0),
	}
}

func (fm *fpeermap) add(hash string, peer models.Peer) {
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

// интерфейс для локального хранилища конкретной ноды
type nodeStorage interface {
	GetManlist() []models.Manifest
	// сохранить на диск в datadir
	Save2disk(src string) (string, int64, error)
	// сохранить в запись бд в нужный бакет
	Save2db(man models.Manifest, bucket string) error
	// список всех хешей (для синхронизации)
	GetHashesList() ([]string, error)
	// получить манифест по его хешу
	Getmanh(hash string, bucket string) (*models.Manifest, error)
	// получить манифест по хешу его файла (тут используется индекс)
	Getmanfh(fhash string, bucket string) (*models.Manifest, error)
	// проверить есть ли такая заметка на всей доске
	HasNote(hash string) bool
	// проверить, есть ли файл на диске (и в бакете local соотв.)
	FileExists(fhash string) bool
	//
	SaveBlob(fhash string, r io.Reader) error
	//
	Fhash2path(fhash string) string
	//
	CleanVirtual() error
	//
	Delfile(fhash string) error
	//
	Delman(hash string, bucket string) (string, error)
	//
	RepubLocal() error
}

func (n *Node) Uploadf(src, title, desc string) (*models.Manifest, error) {
	var fhash string
	var fsize int64
	var err error

	if src != "" {
		fhash, fsize, err = n.Storage.Save2disk(src)
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

	if err = n.Storage.Save2db(*man, models.Bucketvirtual); err != nil {
		log.Printf("failed to sync self to virtual: %v", err)
	}

	// save to db bc we saved it to disk
	if err = n.Storage.Save2db(*man, models.Bucketlocal); err != nil {
		return nil, fmt.Errorf("db err: %w", err)
	}
	return man, nil
}

// receive file from net -> process
func (n *Node) Recvf(p models.Peer, man *models.Manifest) error {
	// 1: to file peers map
	n.filepeers.add(man.FileHash, p)

	// 2: save to 'virtual' bucket ANYWAY \
	// if file is already exists, data will update.
	if err := n.Storage.Save2db(*man, models.Bucketvirtual); err != nil {
		log.Printf("[db] failed to save to virtual: %v", err)
	}

	// 3. file was flushed to disk -> save that to local
	if man.FileSize > 0 && n.Storage.FileExists(man.FileHash) {
		return n.Storage.Save2db(*man, models.Bucketlocal)
	}

	// 4. auto-download blobs under max size limit
	if man.FileSize <= mindlsize && man.FileSize > 0 {
		go func() {
			err := n.DlFile(p, man.FileHash)
			if err == nil {
				n.Storage.Save2db(*man, models.Bucketlocal)
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
// (special endpoint for scalability)
func (n *Node) askpeers(fhash string) {
	n.peers.mu.Lock()
	peers := n.peers.d
	n.peers.mu.Unlock()

	for _, peer := range peers {
		go func(p models.Peer) {
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

// donload file from peer with file hash
func (n *Node) Dlf(fhash string) (io.ReadCloser, error) {
	// 1: ищем локально через Storage
	if n.Storage.FileExists(fhash) {
		return os.Open(n.Storage.Fhash2path(fhash))
	}

	// 2: ищем в пир-мапе
	hostl, ok := n.filepeers.getpeerlist(fhash)
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

func (n *Node) DlFile(p models.Peer, fhash string) error {
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

	if err := n.Storage.SaveBlob(fhash, tr); err != nil {
		return err
	}

	realhash := hex.EncodeToString(h.Sum(nil))
	if fhash != realhash {
		n.Storage.Delfile(fhash)
		return fmt.Errorf("[dl file] hash mismatch: expected %s, got %s", fhash[:8], realhash[:8])
	}

	return nil
}

func (n *Node) RmNote(hash string) error {
	fhash, err := n.Storage.Delman(hash, models.Bucketlocal)
	if err != nil {
		return fmt.Errorf("")
	}

	if _, err := n.Storage.Delman(hash, models.Bucketvirtual); err != nil {
		return fmt.Errorf("")
	}

	if fhash != "" {
		return n.Storage.Delfile(fhash)
	}

	return nil
}
