package network

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

// мапа пиров которые раздают конкретный файл (если его нет у ноды)
// map[id заметки]список_пиров
type fpeermap struct {
	mu sync.Mutex
	d  map[string][]models.Peer
}

func newFpm() *fpeermap {
	return &fpeermap{
		d: make(map[string][]models.Peer, 0),
	}
}

// добавить пир и хеш файла, который он раздает в мапу (для этого лочично нужен манифест и сам пир)
func (fm *fpeermap) Add(hash string, peer models.Peer) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for _, p := range fm.d[hash] {
		if p.UID == peer.UID {
			return
		}
	}

	fm.d[hash] = []models.Peer{peer}
}

// удалить конкретный пир из всех списков (в том случае если он не активен)
func (fm *fpeermap) RmPeer(peer models.Peer) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for _, ps := range fm.d {
		for i, pr := range ps {
			if pr == peer {
				ps = append(ps[:i], ps[i+1:]...)
			}
		}
	}
}

// удалить манифест из карты пиров вместе со всеми пирами (в случае есои из сети исчез сам манифест)
func (fm *fpeermap) RmMan(hash string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	delete(fm.d, hash)
}

// интерфейс для локального хранилища конкретной ноды
type nodeStorage interface {
	// saving files (node.go -> ProcessFile)
	Save2disk(src string) (string, int64, error)
	Save2db(man models.Manifest) error

	// sync files (sync.go)
	GetHashesList() ([]string, error)
	GetFMan(hash string) (*models.Manifest, error)
	HasNote(hash string) bool

	// downloading files (node.go -> DownloadBlob)
	FileExists(fhash string) bool // проверить, есть ли файл на диске
	SaveBlob(fhash string, r io.Reader) error
}

// загрузить файл на доску из локального хранилища
// сырые данные -> сохранение на диск -> подписываем -> сохранение в бд -> готовый manifest
func (n *Node) Savef(src, title, desc, authorname string) (*models.Manifest, error) {
	// сохраняем файл физически на диск
	fhash, fsize, err := n.Storage.Save2disk(src)
	if err != nil {
		return nil, err
	}

	man := models.NewMan(
		title,
		filepath.Base(src),
		desc,
		fhash,
		fsize,
	)

	man.AuthorUID = n.UID
	man.AuthorName = authorname

	if err := man.Sign(n.PrivateK); err != nil {

	}

	man.Hash = hex.EncodeToString(man.CalcID())

	// сохраняем в бд потому что он лежит на диске
	if err = n.Storage.Save2db(*man); err != nil {
		return nil, err
	}

	return man, nil
}

// получить манифест от другой ноды и обработать его
func (n *Node) Recvf(p models.Peer, man *models.Manifest) error {
	// если файл весит меньше 1мб, скачиваем и сохраняем в локальную базу
	if man.Size <= 10*1<<20 && !n.Storage.FileExists(man.FileHash) {
		err := n.DlBlob(p, man.FileHash)
		if err != nil {
			return err
		}

		if err := n.Storage.Save2db(*man); err != nil {
			return err
		}
	}

	n.fpeers.Add(man.FileHash, p)
	// и спрашиваем у других есть ли у них файл с этим хешем

	// когда одна нода делает broadcast, а наша нода его принимает, в логах должно отобразиться
	// [RECV] new manifest received: man.Hash[:8] - available on [список нод где это файл также есть]

	return nil
}

// скачать файл у пира
func (n *Node) DlBlob(p models.Peer, fhash string) error {
	c := http.Client{Timeout: 30 * time.Second}
	resp, err := c.Get(fmt.Sprintf("http://%s:%d/api/dl/%s", p.IP, p.Port, fhash))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer returned status: %d", resp.StatusCode)
	}

	return n.Storage.SaveBlob(fhash, resp.Body)
}
