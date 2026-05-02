// handlers ONLY for frontend endpoints (/api/...)
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

// GET /api/list : получить список всех хешей в локальной сети
func (a *App) listallh(w http.ResponseWriter, r *http.Request) {
	mlist := a.internalst.GetManList()
	json.NewEncoder(w).Encode(mlist)
}

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
	conns := a.Node.GetConnsF()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conns); err != nil {
		log.Printf("[ERR] failed to encode peers: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// GET /api/dl/{hash}
func (a *App) dlhandler(w http.ResponseWriter, r *http.Request) {
	fhash := r.PathValue("hash")
	if fhash == "" {
		http.Error(w, "hash is required", http.StatusBadRequest)
		return
	}

	log.Printf("[ui] download request for hash: %s", fhash)

	err := a.Node.Dlf(fhash)
	if err != nil {
		log.Printf("[ui] download error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("download complete"))
}

// GET /api/del/{hash}
func (a *App) p2pdelhandler(w http.ResponseWriter, r *http.Request) {
	mh := r.PathValue("hash")
	if mh == "" {
		http.Error(w, "hash is required", http.StatusBadRequest)
		return
	}

	log.Printf("[ui] delete request for hash: %s", mh)

	err := a.Node.Delf(mh)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("delete complete"))
}
