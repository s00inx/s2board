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
	Node *network.Node
	st   *storage.Storage
	cfg  *Config
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

	Node, err := network.ConnNode(filepath.Join(st.Dir, "node.key"), a.cfg.Port)
	if err != nil {
		log.Fatal("[FATAL] node connect error: ", err)
	}
	Node.Storage = st

	a.Node = Node

	a.Node.Storage.CleanVirtual()
	a.Node.Storage.RepubLocal()

	liface, ipstr := network.GetLocalIface()
	if liface == nil {
		log.Println("[WARN] could not find valid net interface, using localhost")
	}
	url := fmt.Sprintf("http://%s:%d", ipstr, a.cfg.Port)

	go a.Node.Discover(context.Background()) // поиск соседей

	mdnsrv, err := network.InitMdns(liface, a.Node.UID, a.cfg.Name, a.cfg.Port)
	if err != nil {
		log.Println("[WARN] mDNS registration failed: ", err)
	} else {
		defer mdnsrv.Shutdown()
	}

	fmt.Printf("\nservice is STARTED...\n")
	fmt.Printf("node UID: %s\n", a.Node.UID[:16])
	fmt.Printf("local UI: %s\n", url)
	// network.PrintQr(url)

	mux := a.setupRoutes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", a.cfg.Port), mux))
}

func (a *App) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "tmpstatic/index.html")
	})

	mux.HandleFunc("GET /api/list", a.listallh)

	mux.HandleFunc("GET /api/dl/{hash}", a.dlh)
	mux.HandleFunc("GET /api/hasf/{hash}", a.hasfh)

	mux.HandleFunc("POST /api/recv", a.recvh)

	mux.HandleFunc("POST /api/fetch", a.fetchh)

	mux.HandleFunc("GET /api/hello", a.helloh)
	mux.HandleFunc("GET /api/bye/{peer_id}", a.byeh)

	mux.HandleFunc("POST /api/create", a.createh)
	mux.HandleFunc("POST /api/del", a.delh)

	mux.HandleFunc("GET /api/getpeers", a.getpeersh)

	return mux
}
