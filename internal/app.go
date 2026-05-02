package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/s00inx/s2board/internal/codec"
	"github.com/s00inx/s2board/internal/network"
	"github.com/s00inx/s2board/internal/storage"
	"github.com/s00inx/s2board/internal/transport"
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
	Node.Codec = codec.JSONCodec{}
	Node.InternalStorage = ist
	Node.FileStorage = est
	a.Node = Node

	// cleanup node local databases
	a.Node.InternalStorage.Cleanvb()
	a.Node.InternalStorage.InitLocal()

	// get a net interface to link node <-> interface
	liface, ipstr := network.GetLocalIface()
	if liface == nil {
		log.Println("[WARN] could not find valid net interface, using localhost")
	}
	a.Node.IP = ipstr

	// url := fmt.Sprintf("http://%s:%d", ipstr, a.cfg.Port)
	tr := &transport.HTTPTransport{
		Codec: a.Node.Codec,
		Port:  a.cfg.Port,
	}
	a.Node.Transport = tr

	mux := a.setupRoutes()

	// setup mDns for local node discovery
	_, mdnsrv, err := network.InitMdns(liface, a.Node.UID, a.cfg.Name, a.cfg.Port)
	if err != nil {
		log.Println("[WARN] mDNS registration failed: ", err)
	} else {
		defer mdnsrv.Shutdown()
	}

	// setup node in local network
	go a.Node.Discover(context.Background())

	// print debug information
	// fmt.Printf("node UID: %s | local UI: http://%s:%d\n", a.Node.UID[:16], url, a.cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", a.cfg.Port), mux))
}

func (a *App) setupRoutes() *http.ServeMux {
	mux := a.Node.Transport.Start(a.cfg.Port, a.Node.ProcessPacket)

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "tmpstatic/index.html")
	})

	// main page
	mux.HandleFunc("GET /api/list", a.listallh)

	// FRONTEND simple endpoints
	mux.HandleFunc("GET /api/getpeers", a.getpeersh)
	mux.HandleFunc("GET /api/me", a.meh)
	mux.HandleFunc("POST /api/create", a.createh)
	mux.HandleFunc("GET /api/dl/{hash}", a.dlhandler)
	mux.HandleFunc("GET /api/del/{hash}", a.p2pdelhandler)

	return mux
}
