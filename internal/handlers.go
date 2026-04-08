package internal

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// GET /api/list : получить список всех хешей в локальной сети
func (a *App) listallh(w http.ResponseWriter, r *http.Request) {
	mlist := a.st.GetManlist()

	json.NewEncoder(w).Encode(mlist)
}

// GET /api/hello : получить список всех хешей конкретной ноды
func (a *App) helloh(w http.ResponseWriter, r *http.Request) {
	hashes, err := a.curnode.GetHashes()
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
	err := json.NewDecoder(r.Body).Decode(&hashes)
	if err != nil {
		log.Printf("[ERR] fetch: failed to decode JSON: %v", err)
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	m2fetch, err := a.curnode.FetchManifests(hashes)
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

	content, err := a.curnode.Dlf(hval)
	if err != nil {
		log.Printf("[ERR] Download failed for %s: %v", hval[:8], err)
		http.Error(w, "File not found in network", http.StatusNotFound)
		return
	}
	defer content.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=\""+hval+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	// w.Header().Set("Content-Length")

	_, err = io.Copy(w, content)
	if err != nil {
		log.Printf("[ERR] Stream error for %s: %v", hval[:8], err)
	}
}

// GET /api/hasf/{hash} : выяснить есть ли файл у ноды
func (a *App) hasfh(w http.ResponseWriter, r *http.Request) {
	hashval := r.PathValue("hash")

	if !a.curnode.Storage.FileExists(hashval) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(200)
}

func (a *App) byeh(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("peer_id")

	a.curnode.RmPeer(pid)
	a.curnode.NodeBye(http.Client{})
}

// POST /api/recv : получить манифест
func (a *App) recvh(w http.ResponseWriter, r *http.Request) {
	// var man models.Manifest
	// if err := json.NewDecoder(r.Body).Decode(&man); err != nil {
	// 	http.Error(w, "bad json", 400)
	// 	return
	// }

	// if err := a.st.SaveManifest(man); err != nil {
	// 	http.Error(w, "storage error", 500)
	// 	return
	// }

	// log.Printf("[RECV] new manifest received: %s", man.Hash[:8])
	// w.WriteHeader(http.StatusOK)
}

// POST /api/create : создать новую заметку
func (a *App) createh(w http.ResponseWriter, r *http.Request) {

}

// vibecoded :((
func (a *App) testh(w http.ResponseWriter, r *http.Request) {
	// testFileName := "test.txt"
	// if _, err := os.Stat(testFileName); os.IsNotExist(err) {
	// 	os.WriteFile(testFileName, []byte("kto chitaet krosavchik =))"), 0644)
	// }

	// manb := a.curnode.ProcessFile(testFileName, "test", "sent via p2p")
	// manifest := &models.Manifest{}

	// json.Unmarshal(manb, manifest)
	// go a.curnode.Broadcast(manifest)
}
