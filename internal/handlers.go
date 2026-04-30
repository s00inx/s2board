package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// frontend endpoints
// RU: ниже расписана логика работы и почему была выбрана именно она
// фронтенд просто берет записи у бекенда, так что логично что тут только GET
// это просто костыль для того, чтобы был красивый фронтенд, в идеальной p2p сети такого конечно нет

// GET /api/list : получить список всех хешей в локальной сети
func (a *App) listallh(w http.ResponseWriter, r *http.Request) {
	mlist := a.internalst.GetManList()
	json.NewEncoder(w).Encode(mlist)
}

// // GET /api/dl/{hash} : скачать файл по его хешу
// func (a *App) dlh(w http.ResponseWriter, r *http.Request) {
// 	hval := r.PathValue("hash")
// 	if len(hval) < 8 {
// 		http.Error(w, "invalid hash", http.StatusBadRequest)
// 		return
// 	}

// 	c, err := a.Node.Dlf(hval)
// 	if err != nil {
// 		log.Printf("[ERR] Download failed for %s: %v", hval[:8], err)
// 		http.Error(w, "File not found in network", http.StatusNotFound)
// 		return
// 	}
// 	defer c.Close()

// 	man, _ := a.Node.DbStorage.GetManfh(hval, models.Bucketvirtual)
// 	if man != nil {
// 		w.Header().Set("Content-Disposition", "attachment; filename=\""+url.PathEscape(man.Title)+"\"")
// 		w.Header().Set("Content-Length", fmt.Sprintf("%d", man.FileSize))
// 	} else {
// 		w.Header().Set("Content-Disposition", "attachment; filename=\""+hval+"\"")
// 	}

// 	w.Header().Set("Content-Type", "application/octet-stream")

// 	_, err = io.Copy(w, c)
// 	if err != nil {
// 		log.Printf("[ERR] Stream error for %s: %v", hval[:8], err)
// 	}
// }

// GET /api/me
func (a *App) meh(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"uid":  a.Node.UID,
		"name": a.Node.PubName,
	})
}

func (a *App) createh(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "form error", 400)
		return
	}

	title := r.FormValue("title")
	desc := r.FormValue("desc")

	var tempPath string
	file, header, err := r.FormFile("file")

	if err == nil {
		defer file.Close()
		tempPath = filepath.Join(os.TempDir(), header.Filename)
		out, err := os.Create(tempPath)
		if err != nil {
			http.Error(w, "temp file creation failed", 500)
			return
		}
		io.Copy(out, file)
		out.Close()
		defer os.Remove(tempPath)
	} else if err != http.ErrMissingFile {
		fmt.Print(err)
		http.Error(w, "file error", 400)
		return
	}

	man, err := a.Node.Createf(tempPath, title, desc)
	if err != nil {
		log.Printf("[ERR] upload failed: %v", err)
		http.Error(w, "internal p2p error", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(man)
}

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

// key simplification ONLY for http : GET /dl/{filehash}
// func (a *App) p2pdlhandler(w http.ResponseWriter, r *http.Request) {
// 	hval := r.PathValue("hash")
// 	if len(hval) < 8 {
// 		http.Error(w, "invalid hash", http.StatusBadRequest)
// 		return
// 	}

// 	c, err := a.Node.Dlf(hval)
// 	if err != nil {
// 		log.Printf("[ERR] Download failed for %s: %v", hval[:8], err)
// 		http.Error(w, "File not found in network", http.StatusNotFound)
// 		return
// 	}
// 	defer c.Close()

// 	man, _ := a.Node.DbStorage.GetManfh(hval, models.Bucketvirtual)
// 	if man != nil {
// 		w.Header().Set("Content-Disposition", "attachment; filename=\""+url.PathEscape(man.Title)+"\"")
// 		w.Header().Set("Content-Length", fmt.Sprintf("%d", man.FileSize))
// 	} else {
// 		w.Header().Set("Content-Disposition", "attachment; filename=\""+hval+"\"")
// 	}

// 	w.Header().Set("Content-Type", "application/octet-stream")

// 	_, err = io.Copy(w, c)
// 	if err != nil {
// 		log.Printf("[ERR] Stream error for %s: %v", hval[:8], err)
// 	}
// }
