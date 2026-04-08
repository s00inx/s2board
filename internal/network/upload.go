package network

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/s00inx/s2board/internal/models"
)

// мапа пиров которые раздают конкретный файл (если его нет у ноды)
// map[хкш файла]список_id_пиров (сделано потому что мы имеем дело с реальными файлами на диске, так зачем давать к ним лоступ по айди манифеста?)
type fpeermap struct {
	mu sync.Mutex
	d  map[string][]string
}

func newFpm() *fpeermap {
	return &fpeermap{
		d: make(map[string][]string, 0),
	}
}

// добавить пир и хеш файла, который он раздает в мапу (для этого лочично нужен манифест и сам пир)
func (fm *fpeermap) add(hash string, peer models.Peer) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for _, p := range fm.d[hash] {
		if p == peer.UID {
			return
		}
	}

	fm.d[hash] = append(fm.d[hash], peer.UID)
}

// удалить конкретный пир из всех списков (в том случае если он не активен)
func (fm *fpeermap) rmPeer(peerid string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for _, ps := range fm.d {
		for i, pr := range ps {
			if pr == peerid {
				ps = append(ps[:i], ps[i+1:]...)
			}
		}
	}
}

// удалить манифест из карты пиров вместе со всеми пирами (в случае есои из сети исчез сам манифест)
func (fm *fpeermap) rmMan(hash string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	delete(fm.d, hash)
}

func (fm *fpeermap) getpeerlist(hash string) ([]string, bool) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	h, ok := fm.d[hash]
	return h, ok
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

	n.fpeers.add(man.FileHash, p)
	// и спрашиваем у других есть ли у них файл с этим хешем

	// когда одна нода делает broadcast, а наша нода его принимает, в логах должно отобразиться
	// [RECV] new manifest received: man.Hash[:8] - available on [список нод где это файл также есть]

	return nil
}

// скачать файл у пира по его хешу (в несколько этапов)
// активируется при вызове хендлера скачивания (/api/dl) \ принять хеш -> выдать ридер с самим файлом
// (файл если он физически на ноде, либо потом если текуцщая нода выступает как прокси)
func (n *Node) Dlf(fhash string) (io.ReadCloser, error) {
	// 1: ищем в своем хранилище
	if n.Storage.FileExists(fhash) {
		return os.Open(filepath.Join(fhash[:2], fhash))
	}

	// 2: ищем этот хеш в пир мапе
	c := &http.Client{Timeout: 10 * time.Second}
	// ^-- todo: использовать общего клиента

	var err error
	if hostl, ok := n.fpeers.getpeerlist(fhash); ok {
		for _, fpeerid := range hostl {
			fpeer, ok := n.peermap.d[fpeerid]
			if !ok {
				continue
			}

			dsturlp := fmt.Sprintf("http://%s:%d/api", fpeer.IP, fpeer.Port)

			hfr, err := c.Get(dsturlp + "/hasf/" + fhash)
			if err != nil {
				n.RmPeer(fpeerid)
			}

			if hfr.StatusCode == http.StatusOK {
				r, err := c.Get(dsturlp + "/dl/" + fhash)
				if err != nil {
					hfr.Body.Close()
					continue
				}
				hfr.Body.Close()
				return r.Body, err
			} else {
				hfr.Body.Close()
				continue
			}
		}
	}

	// 3: если до сюда дошло - файл не скачался -> ошибка
	return nil, err
}

// скачать файл у пира
func (n *Node) DlBlob(p models.Peer, fhash string) error {
	return nil
}
