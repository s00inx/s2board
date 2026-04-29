package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/s00inx/s2board/internal/models"
)

// frontend endpoints
// RU: ниже расписана логика работы и почему была выбрана именно она
// фронтенд просто берет записи у бекенда, так что логично что тут только GET

// GET /api/list : получить список всех хешей в локальной сети
func (a *App) listallh(w http.ResponseWriter, r *http.Request) {
	mlist := a.internalst.GetManList()
	json.NewEncoder(w).Encode(mlist)
}

// GET /api/hello : получить список всех хешей конкретной ноды
func (a *App) helloh(w http.ResponseWriter, r *http.Request) {
	hashes, err := a.Node.GetHashes()
	if err != nil {
		http.Error(w, "", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hashes)
}

// POST /api/fetch : получить список манифестов по списку хешей
func (a *App) fetchh(w http.ResponseWriter, r *http.Request) {
	var hashes []string
	if err := json.NewDecoder(r.Body).Decode(&hashes); err != nil {
		log.Printf("[ERR] fetch: failed to decode JSON: %v", err)
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	m2fetch, err := a.Node.FetchManifests(hashes)
	if err != nil {
		log.Printf("[ERR] fetch: db error: %v", err)
		http.Error(w, "internal database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(m2fetch); err != nil {
		log.Printf("[ERR] fetch: failed to encode response: %v", err)
	}
}

// GET /api/dl/{hash} : скачать файл по его хешу
func (a *App) dlh(w http.ResponseWriter, r *http.Request) {
	hval := r.PathValue("hash")
	if len(hval) < 8 {
		http.Error(w, "invalid hash", http.StatusBadRequest)
		return
	}

	c, err := a.Node.Dlf(hval)
	if err != nil {
		log.Printf("[ERR] Download failed for %s: %v", hval[:8], err)
		http.Error(w, "File not found in network", http.StatusNotFound)
		return
	}
	defer c.Close()

	man, _ := a.Node.DbStorage.GetManfh(hval, models.Bucketvirtual)
	if man != nil {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+url.PathEscape(man.Title)+"\"")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", man.FileSize))
	} else {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+hval+"\"")
	}

	w.Header().Set("Content-Type", "application/octet-stream")

	_, err = io.Copy(w, c)
	if err != nil {
		log.Printf("[ERR] Stream error for %s: %v", hval[:8], err)
	}
}

// GET /api/hasf/{hash} : выяснить есть ли файл у ноды
func (a *App) hasfh(w http.ResponseWriter, r *http.Request) {
	hashval := r.PathValue("hash")

	if !a.Node.FileStorage.FileExists(hashval) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(200)
}

// GET /api/me
func (a *App) meh(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"uid":  a.Node.UID,
		"name": a.Node.PubName,
	})
}

// POST /api/create : создать новую заметку (multipart form)
func (a *App) createh(w http.ResponseWriter, r *http.Request) {
	// if err := r.ParseMultipartForm(32 << 20); err != nil {
	// 	http.Error(w, "form error", http.StatusBadRequest)
	// 	return
	// }

	// title := r.FormValue("title")
	// desc := r.FormValue("desc")
	// author := r.FormValue("author")

	// if author == "" {
	// 	author = a.cfg.Name
	// }

	// // var man *models.Manifest
	// var err error

	// file, header, fileErr := r.FormFile("file")

	// switch fileErr {
	// case nil:
	// 	defer file.Close()
	// 	tempPath := filepath.Join(os.TempDir(), header.Filename)
	// 	out, err := os.Create(tempPath)
	// 	if err != nil {
	// 		http.Error(w, "internal error", 500)
	// 		return
	// 	}
	// 	io.Copy(out, file)
	// 	out.Close()
	// 	defer os.Remove(tempPath)

	// 	// man, err = a.Node.Uploadf(tempPath, title, desc)
	// case http.ErrMissingFile:
	// 	// man, err = a.Node.Uploadf("", title, desc)
	// default:
	// 	http.Error(w, "bad file", http.StatusBadRequest)
	// 	return
	// }

	// if err != nil {
	// 	log.Printf("[ERR] create failed: %v", err)
	// 	http.Error(w, "failed to create manifest", 500)
	// 	return
	// }

	// P2PPacket := models.NewPacket(man, models.Actsave, a.Node.UID, a.Node.PrivateK)
	// go a.Node.Broadcast(P2PPacket)

	// w.WriteHeader(http.StatusOK)
}

// POST /api/del : фронтенд просит удалить манифест
func (a *App) delh(w http.ResponseWriter, r *http.Request) {
	// var req struct {
	// 	Mhash string `json:"hash"`
	// }

	// if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
	// 	http.Error(w, "invalid JSON", http.StatusBadRequest)
	// 	return
	// }
	// defer r.Body.Close()

	// man, err := a.Node.DbStorage.GetManh(req.Mhash, models.Bucketvirtual)
	// if err != nil || man == nil {
	// 	http.Error(w, "manifest not found", http.StatusNotFound)
	// 	return
	// }

	// P2PPacket := models.NewPacket(man, models.Actsave, a.Node.UID, a.Node.PrivateK)
	// go a.Node.Broadcast(P2PPacket)

	// err = a.Node.RmNote(req.Mhash)
	// if err != nil {
	// 	log.Printf("[del] error removing %s: %v", req.Mhash[:8], err)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
}

// GET /api/getpeers : посмотреть список всех пиров в локальной сети
func (a *App) getpeersh(w http.ResponseWriter, r *http.Request) {
	conns := a.Node.GetConns()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conns); err != nil {
		log.Printf("[ERR] failed to encode peers: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// p2p endpoint
// RU: в mvp для транспорта был выбран именно HTTP потому что он прост в отладке, но влечет за собой оверхед
// на заголовки и json, также я использовал только Push на бекенде, Pull оставил только для фронтенда,
// это просто упрощение, в p2p сетях лучше использовать только Push!
// за всю п2п логику отвечает именно 1 эндпоинт /api/p2p, только POST (нет смысла следовать rest)
// todo: сделать бинарный протокол поверх udp

// all p2p Push logic
func (a *App) p2phandler(w http.ResponseWriter, r *http.Request) {
	// decode p2p packet from request
	var incp models.P2PPacket

	if err := json.NewDecoder(r.Body).Decode(&incp); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !incp.Verify() {
		log.Printf("[p2p] invalid signature from %s -> denied", incp.Senderuid[:8])
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// process packet payload based on action
	switch incp.Action {
	// handshake logic : recv hello -> send helloack
	case models.ActHello:
		pl, err := a.Node.RecvHellof(&incp, a.Node.IP, a.Node.Port)
		if err != nil {
			return
		}

		ack := models.NewPacket(pl, models.ActHelloAck, a.Node.UID, a.Node.PrivateK)
		json.NewEncoder(w).Encode(ack)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
