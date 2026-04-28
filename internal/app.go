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

type Config struct {
	DataDir string
	Port    int
	Name    string
}

type App struct {
	Node       *network.Node
	internalst *storage.InternalStorage
	extst      *storage.ExternalStorage
	cfg        *Config
}

func NewApp(cfg *Config) *App {
	return &App{
		cfg: cfg,
	}
}

func (a *App) Run() {
	// init storage, find or create deps
	ist, est, err := storage.Init(a.cfg.DataDir)
	if err != nil {
		log.Fatal("storage init error: ", err)
	}
	a.internalst = ist
	a.extst = est

	// setup node on this device
	Node, err := network.ConnNode(filepath.Join(est.Dir, "node.key"), a.cfg.Port, a.cfg.Name)
	if err != nil {
		log.Fatal("[FATAL] node connect error: ", err)
	}
	Node.DbStorage = ist
	Node.FileStorage = est
	a.Node = Node

	// cleanup node local databases
	a.Node.DbStorage.Cleanvb()
	a.Node.DbStorage.InitLocal()

	// get a net interface to link node <-> interface
	liface, ipstr := network.GetLocalIface()
	if liface == nil {
		log.Println("[WARN] could not find valid net interface, using localhost")
	}

	// setup mDns for local node discovery
	nm, mdnsrv, err := network.InitMdns(liface, a.Node.UID, a.cfg.Name, a.cfg.Port)
	if err != nil {
		log.Println("[WARN] mDNS registration failed: ", err)
	} else {
		defer mdnsrv.Shutdown()
	}

	// setup node in local network
	go a.Node.Discover(context.Background())

	// print debug information
	url := fmt.Sprintf("http://%s:%d", ipstr, a.cfg.Port)
	fmt.Printf("\nservice is STARTED...\n")
	fmt.Printf("node UID: %s\n", a.Node.UID[:16])
	fmt.Printf("local UI: %s\n", url)
	fmt.Printf("local IP: %s:%d\n", nm, a.cfg.Port)
	mux := a.setupRoutes()

	// up node server on :port for routing (http REST api)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", a.cfg.Port), mux))
}

func (a *App) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "tmpstatic/index.html")
	})

	// main page
	mux.HandleFunc("GET /api/list", a.listallh)

	// downloading files
	mux.HandleFunc("GET /api/dl/{hash}", a.dlh)
	mux.HandleFunc("GET /api/hasf/{hash}", a.hasfh)

	// sync and sending
	mux.HandleFunc("POST /api/fetch", a.fetchh)
	mux.HandleFunc("GET /api/hello", a.helloh)

	// p2p network
	mux.HandleFunc("POST /api/p2p", a.p2phandler)

	// posting files
	mux.HandleFunc("POST /api/create", a.createh)
	mux.HandleFunc("POST /api/del", a.delh)

	// etc ..
	mux.HandleFunc("GET /api/getpeers", a.getpeersh)
	mux.HandleFunc("GET /api/me", a.meh)

	return mux
}
