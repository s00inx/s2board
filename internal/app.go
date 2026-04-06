package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/s00inx/s2board/internal/network"
	"github.com/s00inx/s2board/internal/storage"
)

// конфиг это абстракция для удобной настройки приложения
type Config struct {
	DataDir string // где лежат бд и блобы
	Port    int    // порт веб-сервера
	Name    string // имя ноды (по желанию)
}

// главная абстаркция над всем сервисом, инкапсулирует логику ноды и хранилища
type App struct {
	curnode *network.Node
	st      *storage.Storage
	cfg     *Config
}

// создает экземпляр приложения и подготавливает зависимости
func NewApp(cfg *Config) *App {
	return &App{
		cfg: cfg,
	}
}

// точка входа, что запускает все сетевые и системные процессы
func (a *App) Run() {
	st, err := storage.Init(a.cfg.DataDir)
	if err != nil {
		log.Fatal("storage init error: ", err)
	}
	a.st = st

	curnode, err := network.NodeConnect(filepath.Join(st.Dir, "node.key"), a.cfg.Port)
	if err != nil {
		log.Fatal("[FATAL] node connect error: ", err)
	}
	curnode.Storage = st
	a.curnode = curnode

	liface, ipstr := network.GetLocalIface()
	if liface == nil {
		log.Println("[WARN] could not find valid net interface, using localhost")
	}
	url := fmt.Sprintf("http://%s:%d", ipstr, a.cfg.Port)

	go a.curnode.Discover(context.Background()) // поиск соседей

	mdnsrv, err := network.InitMdns(liface, a.curnode.UID, a.cfg.Port)
	if err != nil {
		log.Println("[WARN] mDNS registration failed: ", err)
	} else {
		defer mdnsrv.Shutdown()
	}

	fmt.Printf("\nservice is STARTED...\n")
	fmt.Printf("node UID: %s\n", a.curnode.UID[:16])
	fmt.Printf("local UI: %s\n", url)
	network.PrintQr(url)

	mux := a.setupRoutes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", a.cfg.Port), mux))
}

func (a *App) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Главная
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[NETWORK] new client conn: %s", r.RemoteAddr)

		notes, _ := a.st.GetHashes()
		fmt.Fprintf(w, "S2BOARD Active. Notes count: %d", len(notes))
	})

	// mux.HandleFunc("GET /api/notes", a.listNotesHandler) // список всех заметок
	mux.HandleFunc("GET /api/dl/{hash}", a.dlHandler) // скачать файл
	mux.HandleFunc("POST /api/recv", a.recvHandler)

	mux.HandleFunc("GET /api/sync", a.curnode.GetHashes)
	mux.HandleFunc("POST /api/sync/fetch", a.curnode.FetchManifests)

	mux.HandleFunc("GET /api/peers", a.curnode.GetPeersHandler)

	mux.HandleFunc("POST /api/test", a.createTestNoteHandler)

	return mux
}
