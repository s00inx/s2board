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
	"sync"

	"github.com/s00inx/s2board/internal/models"
)

const (
	mindlsize = 25 * 1 << 20
)

// мапа пиров которые раздают конкретный файл (если его нет у ноды)
// map[хкш файла]список_id_пиров (сделано потому что мы имеем дело с реальными файлами на диске, так зачем давать к ним лоступ по айди манифеста?)
type fpeermap struct {
	mu sync.Mutex
	d  map[string][]string
}

func newfilepeermap() *fpeermap {
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
func (fm *fpeermap) rmpeer(peerid string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for hash, ps := range fm.d {
		nps := ps[:0]
		for _, pr := range ps {
			if pr != peerid {
				nps = append(nps, pr)
			}
		}
		fm.d[hash] = nps

		if len(nps) == 0 {
			delete(fm.d, hash)
		}
	}
}

// удалить манифест из карты пиров вместе со всеми пирами (в случае если из сети исчез сам манифест)
func (fm *fpeermap) rmman(hash string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	delete(fm.d, hash)
}

// посмотреть значение по ключу (и есть ли оно вообще) потокобезопасно
func (fm *fpeermap) getpeerlist(hash string) ([]string, bool) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	h, ok := fm.d[hash]
	return h, ok
}

// интерфейс для локального хранилища конкретной ноды
type nodeStorage interface {
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

// загрузить файл на доску из локального хранилища
// сырые данные -> сохранение на диск -> подписываем -> сохранение в бд -> готовый manifest
func (n *Node) Uploadf(src, title, desc, authorname string) (*models.Manifest, error) {
	// сохраняем файл физически на диск
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
		filepath.Base(src),
		desc,
		fhash,
		fsize,
	)
	man.AuthorUID = n.UID
	man.AuthorName = authorname

	if err := man.Sign(n.PrivateK); err != nil {
		return nil, fmt.Errorf("failed to sign manifest: %w", err)
	}

	man.Hash = hex.EncodeToString(man.CalcID())

	// сохраняем в бд потому что он лежит на диске
	if err = n.Storage.Save2db(*man, "local"); err != nil {
		return nil, fmt.Errorf("db err: %w", err)
	}

	// свои посты тоже должны быть в ленте
	if err = n.Storage.Save2db(*man, "virtual"); err != nil {
		log.Printf("failed to sync self to virtual: %v", err)
	}

	return man, nil
}

// получить файл -> обработать по логике
func (n *Node) Recvf(p models.Peer, man *models.Manifest) error {
	// 1: в карту пиров (теперь мы знаем, у кого просить)
	n.fpeers.add(man.FileHash, p)

	// 2: в ЛЮБОМ случае сохраняем в virtual \
	// если такой манифест уже был, он просто обновится (например, список пиров расширится)
	if err := n.Storage.Save2db(*man, "virtual"); err != nil {
		log.Printf("[DB] failed to save to virtual: %v", err)
	}

	// 3. если файл уже есть на диске, пометим его и в local
	if n.Storage.FileExists(man.FileHash) {
		return n.Storage.Save2db(*man, "local")
	}

	if man.Size <= mindlsize {
		go func() {
			err := n.DlBlob(p, man.FileHash)
			if err == nil {
				n.Storage.Save2db(*man, "local")
				log.Printf("[AUTO-DL] success: %s", man.FileHash[:8])
			}
		}()
	}

	go n.askpeers(man.FileHash)

	return nil
}

// когда файл приходит на ноду, она составляет полную карту пиров, спрашивает всех пиров в сети, есть ли у них этот файл.
// это нужно для отказоустойчивости (и чтобы сделать загрузку чанками в будущем:))
func (n *Node) askpeers(fhash string) {
	n.peermap.mu.Lock()
	peers := n.peermap.d
	n.peermap.mu.Unlock()

	for _, peer := range peers {
		go func(p models.Peer) {
			url := fmt.Sprintf("http://%s:%d/api/hasf/%s", p.IP, p.Port, fhash)
			resp, err := n.client.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				n.fpeers.add(fhash, p)
				log.Printf("[FOUND] File %s also available on %s", fhash[:8], p.IP)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}(peer)
	}
}

// скачать файл у пира по его хешу (в несколько этапов)
// активируется при вызове хендлера скачивания (/api/dl) \ принять хеш -> выдать ридер с самим файлом
// (файл если он физически на ноде, либо потом если текущая нода выступает как прокси)
func (n *Node) Dlf(fhash string) (io.ReadCloser, error) {
	// 1: ищем локально через Storage
	if n.Storage.FileExists(fhash) {
		return os.Open(n.Storage.Fhash2path(fhash))
	}

	// 2: ищем в пир-мапе
	hostl, ok := n.fpeers.getpeerlist(fhash)
	if !ok || len(hostl) == 0 {
		return nil, fmt.Errorf("no peers found for hash %s", fhash[:8])
	}

	for _, fpeerid := range hostl {
		fpeer, ok := n.peermap.d[fpeerid]
		if !ok {
			continue
		}

		dsturlp := fmt.Sprintf("http://%s:%d/api/dl/%s", fpeer.IP, fpeer.Port, fhash)

		resp, err := n.client.Get(dsturlp)
		if err != nil {
			log.Printf("[PROXY] peer %s error: %v", fpeerid[:8], err)
			n.RmPeer(fpeerid)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}

		resp.Body.Close()
	}

	return nil, fmt.Errorf("failed to retrieve file from %d peers", len(hostl))
}

// скачать файл у конкретного пира (и все, дальнейшим уже занимаются другие функции)
func (n *Node) DlBlob(p models.Peer, fhash string) error {
	dsturl := fmt.Sprintf("http://%s:%d/api/dl/%s", p.IP, p.Port, fhash)

	resp, err := n.client.Get(dsturl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer returned status: %d", resp.StatusCode)
	}

	h := sha256.New()
	tr := io.TeeReader(resp.Body, h)

	if err := n.Storage.SaveBlob(fhash, tr); err != nil {
		return err
	}

	realhash := hex.EncodeToString(h.Sum(nil))
	if fhash != realhash {
		n.Storage.Delfile(fhash)
		return fmt.Errorf("hash mismatch: expected %s, got %s", fhash[:8], realhash[:8])
	}

	return nil
}

// удаляем заметку с доски и из памяти
func (n *Node) RmNote(hash string) error {
	fhash, err := n.Storage.Delman(hash, "local")
	if err != nil {
		return fmt.Errorf("")
	}

	if _, err := n.Storage.Delman(hash, "virtual"); err != nil {
		return fmt.Errorf("")
	}

	if fhash != "" {
		return n.Storage.Delfile(fhash)
	}

	return nil
}
