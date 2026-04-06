package internal

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/s00inx/s2board/internal/models"
)

func (a *App) dlHandler(w http.ResponseWriter, r *http.Request) {
	hval := r.PathValue("hash")
	if len(hval) < 2 {
		http.Error(w, "invalid hash", 400)
		return
	}
	path := filepath.Join(a.cfg.DataDir, "blobs", hval[:2], hval)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+hval+"\"")
	http.ServeFile(w, r, path)
}

func (a *App) recvHandler(w http.ResponseWriter, r *http.Request) {
	var man models.NoteManifest
	if err := json.NewDecoder(r.Body).Decode(&man); err != nil {
		http.Error(w, "bad json", 400)
		return
	}

	if err := a.st.SaveFile(man); err != nil {
		http.Error(w, "storage error", 500)
		return
	}

	log.Printf("[RECV] new manifest received: %s", man.Hash[:8])
	w.WriteHeader(http.StatusOK)
}

func (a *App) createTestNoteHandler(w http.ResponseWriter, r *http.Request) {
	testFileName := "test.txt"
	if _, err := os.Stat(testFileName); os.IsNotExist(err) {
		os.WriteFile(testFileName, []byte("kto chitaet krosavchik =))"), 0644)
	}

	manb := a.curnode.ProcessFile(testFileName, "test", "sent via p2p")
	manifest := &models.NoteManifest{}

	json.Unmarshal(manb, manifest)
	go a.curnode.Broadcast(manifest)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"hash":   manifest.Hash,
		"note":   manifest.Title,
	})
}
