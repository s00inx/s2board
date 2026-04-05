package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/s00inx/stdesk/internal/network"
	"github.com/s00inx/stdesk/internal/storage"
)

// конфиг это абстракция для удобной настройки приложения
type Config struct {
	DataDir string // где лежат бд и блобы
	Port    string // порт веб-сервера
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

	curnode, err := network.NodeConnect(filepath.Join(st.Dir, "node.key"))
	if err != nil {
		log.Fatal("[FATAL] node connect error: ", err)
	}
	curnode.Storage = st
	a.curnode = curnode

	liface, ipstr := network.GetLocalIface()
	if liface == nil {
		log.Println("[WARN] could not find valid net interface, using localhost")
	}
	url := fmt.Sprintf("http://%s:%s", ipstr, a.cfg.Port)

	go a.curnode.Discover(context.Background()) // поиск соседей
	// a.startBackgroundSync()

	mdnsrv, err := network.InitMdns(liface, a.curnode.UID)
	if err != nil {
		log.Println("[WARN] mDNS registration failed: ", err)
	} else {
		defer mdnsrv.Shutdown()
	}

	fmt.Printf("\nservice is STARTED...s\n")
	fmt.Printf("node UID: %s\n", a.curnode.UID[:16])
	fmt.Printf("local UI: %s\n", url)
	network.PrintQr(url)

	mux := a.setupRoutes()
	log.Fatal(http.ListenAndServe(":"+a.cfg.Port, mux))
}

// отдельный метод который регает все апи-ручки
func (a *App) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// homepage
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		finaljson := a.curnode.ProcessFile("test.txt", "Auto-Note", "Synced via P2P")
		w.Write([]byte("S2BOARD Active\n\n" + string(finaljson)))
	})

	// download file via hash
	mux.HandleFunc("GET /api/dl/{hash}", a.dlHandler)

	// receive updates
	mux.HandleFunc("POST /api/recv", a.receiveHandler)

	// delta-sync for nodes
	mux.HandleFunc("GET /api/sync", a.curnode.GetHashes)
	mux.HandleFunc("POST /api/sync/fetch", a.curnode.FetchManifests)

	return mux
}

// startBackgroundSync запускает бесконечный цикл опроса соседей
// func (a *App) startBackgroundSync() {
// 	go func() {
// 		ticker := time.NewTicker(45 * time.Second)
// 		for range ticker.C {
// 			peers := a.curnode.GetConns()
// 			if len(peers) == 0 {
// 				continue
// 			}
// 			log.Printf("[SYNC] checking %d peers for updates...", len(peers))
// 		}
// 	}()
// }
