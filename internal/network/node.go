// центральный узел сети
package network

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/s00inx/stdesk/internal/models"
)

/*
	POST /api/create (json) --> some logic --> ProcessFile
*/

// структура для именно этого устройства в сети
type Node struct {
	// публичная инфа
	Name string // короткое имя для удобства (потом)

	// ключи
	PublicK  ed25519.PublicKey
	PrivateK ed25519.PrivateKey
	UID      string // строковое представление публичного ключа

	Storage nodeStorage

	// параметры конкретного экземпляра
	iface *net.Interface
	ip    [4]byte
	port  int
	peers PeerMap
}

type nodeStorage interface {
	// saving files (node.go -> ProcessFile)
	RegisterFile(src string) (string, int64, error)
	SaveFile(man models.NoteManifest) error

	// sync files (sync.go)
	GetHashes() ([]string, error)
	GetManifest(hash string) (*models.NoteManifest, error)

	// downloading files (node/node.go -> DownloadBlob)
	FileExists(fhash string) bool // проверить, есть ли файл на диске
	SaveBlob(fhash string, r io.Reader) error
}

// подключение ноды к локальной сети
func NodeConnect(privpath string) (*Node, error) {
	f, err := os.ReadFile(privpath)

	if err != nil {
		return mknode(privpath)
	}

	priv := ed25519.PrivateKey(f)
	pub := priv.Public().(ed25519.PublicKey)

	return &Node{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
		peers:    *NewPeerMap(),
	}, nil
}

// заполнить форму на фронтенде -> добавить файл в бд -> сохранить -> вернуть json
func (n *Node) ProcessFile(src, title, desc string) []byte {
	fhash, fsize, err := n.Storage.RegisterFile(src)
	if err != nil {
		return []byte{}
	}

	man := models.NewNote(
		title,
		filepath.Base(src),
		desc,
		fhash,
		fsize,
		models.FileType,
	)

	man.AuthorUID = n.UID
	man.Sign(n.PrivateK)
	man.Hash = hex.EncodeToString(man.CalcId())

	n.Storage.SaveFile(*man)

	manjson, _ := json.MarshalIndent(man, " ", " ")
	return manjson
}

// запустить фоновый процесс синхронизации
func (n *Node) StartSync() {
	// ставим сюда 30 сек
	t := time.NewTicker(30 * time.Second)

	go func() {
		for range t.C {
			ps := n.GetConns()
			if len(ps) == 0 {
				continue
			}

			for _, p := range ps {
				err := n.Syncw(p)
				if err != nil {
					log.Printf("[SYNC] failed with %s: %v", p.UID[:8], err)
				}
			}
		}
	}()
}

// скачать физический файл у пира
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

// вспомогательная функция для того инициализации узла
func mknode(dst string) (*Node, error) {
	// генерим ключи
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(dst, priv, 0600)
	if err != nil {
		return nil, err
	}
	// ^-- сохраняем приватный ключ в указанную директорию

	return &Node{
		PublicK:  pub,
		PrivateK: priv,
		UID:      hex.EncodeToString(pub),
	}, nil
}
