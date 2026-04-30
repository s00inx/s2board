package network

import (
	"io"

	"github.com/s00inx/s2board/internal/models"
)

const (
	mindlsize = 1
	// mindlsize = 25 * 1 << 20
)

// INTERNAL storage interface.
// all interactions with internal db (bbolt, sql etc.)
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

// external storage interface
// all interaction with physical disk
type nodeExternalStorage interface {
	Save2disk(src string) (string, int64, error)
	FileExists(fhash string) bool
	SaveFile(fhash string, r io.Reader) error
	Fhash2path(fhash string) string
	DeleteFile(fhash string) error
}

// // download file from peer using only file hash
// func (n *Node) Dlf(fhash string) (io.ReadCloser, error) {
// 	if n.FileStorage.FileExists(fhash) {
// 		return os.Open(n.FileStorage.Fhash2path(fhash))
// 	}

// 	hostl, ok := n.filepeers.getpeerlist(fhash)

// 	fmt.Println(hostl)
// 	if !ok || len(hostl) == 0 {
// 		return nil, fmt.Errorf("no peers found for hash %s", fhash[:8])
// 	}

// 	for _, fpeerid := range hostl {
// 		fpeer, ok := n.peers.d[fpeerid]
// 		if !ok {
// 			continue
// 		}

// 		dsturlp := fmt.Sprintf("http://%s:%d/api/dl/%s", fpeer.IP, fpeer.Port, fhash)

// 		resp, err := n.client.Get(dsturlp)
// 		if err != nil {
// 			log.Printf("[proxy] peer %s error: %v", fpeerid[:8], err)
// 			n.ForgetPeer(fpeerid)
// 			continue
// 		}

// 		if resp.StatusCode == http.StatusOK {
// 			return resp.Body, nil
// 		}

// 		resp.Body.Close()
// 	}

// 	return nil, fmt.Errorf("failed to retrieve file from %d peers", len(hostl))
// }

// func (n *Node) DlFile(p Peer, fhash string) error {
// 	dsturl := fmt.Sprintf("http://%s:%d/api/dl/%s", p.IP, p.Port, fhash)

// 	resp, err := n.client.Get(dsturl)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return fmt.Errorf("[dl file] peer returned status: %d", resp.StatusCode)
// 	}

// 	h := sha256.New()
// 	tr := io.TeeReader(resp.Body, h)

// 	if err := n.FileStorage.SaveFile(fhash, tr); err != nil {
// 		return err
// 	}

// 	realhash := hex.EncodeToString(h.Sum(nil))
// 	if fhash != realhash {
// 		n.FileStorage.DeleteFile(fhash)
// 		return fmt.Errorf("[dl file] hash mismatch: expected %s, got %s", fhash[:8], realhash[:8])
// 	}

// 	return nil
// }
