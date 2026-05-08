package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

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
	wg         sync.WaitGroup
}

func NewApp(cfg *Config) *App {
	return &App{
		cfg: cfg,
	}
}

func (a *App) Run(ctx context.Context) {
	ist, est, err := storage.Init(a.cfg.DataDir)
	if err != nil {
		log.Fatalf("[FATAL] storage init error: %v", err)
	}
	a.internalst = ist
	a.extst = est

	node, err := network.ConnNode(filepath.Join(est.Dir, "node.key"), a.cfg.Port, a.cfg.Name)
	if err != nil {
		log.Fatalf("[FATAL] node connect error: %v", err)
	}

	node.Codec = codec.JSONCodec{}
	node.InternalStorage = ist
	node.FileStorage = est
	a.Node = node

	a.Node.InternalStorage.Cleanvb()
	a.Node.InternalStorage.InitLocal()

	liface, ipstr := network.GetLocalIface()
	if liface == nil {
		log.Println("[WARN] could not find valid net interface, using localhost")
	}
	a.Node.IP = ipstr

	tr := &transport.HTTPTransport{
		Codec: a.Node.Codec,
		Port:  a.cfg.Port,
	}
	a.Node.Transport = tr

	// setup http server
	mux := a.setupRoutes()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.Port),
		Handler: mux,
	}

	_, mdnsrv, err := network.InitMdns(liface, a.Node.UID, a.cfg.Name, a.cfg.Port)
	if err != nil {
		log.Println("[WARN] mDNS registration failed:", err)
	}

	// monitor for graceful shutdown
	a.wg.Go(func() {
		// waiting for context cancel
		<-ctx.Done()
		// p2p-broadcast packets
		if a.Node != nil {
			a.Node.Byew()

			// timne for broadcast, node don't control that, only push and leave
			time.Sleep(1 * time.Second)
		}

		// zeroconf shutdown (special udp packet)
		if mdnsrv != nil {
			log.Printf("[zeroconf] shutting down service...")
			mdnsrv.Shutdown()
		}

		// context for server shutdown timeout
		shc, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shc)

		log.Printf("[shutdown] all is correct -> exit")
	})

	// start mdns discovery
	a.wg.Go(func() {
		log.Println("[net] starting discovery...")
		a.Node.Discover(ctx)
	})

	a.wg.Go(func() {
		log.Printf("[http] server listening on :%d", a.cfg.Port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("[ERROR] http server error: %v", err)
		}
	})
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

	mux.HandleFunc("GET /api/mpeers", func(w http.ResponseWriter, r *http.Request) {
		a.Node.Codec.EncodeStream(w, a.Node.GetMpeers())
	})

	return mux
}

func (a *App) Wait() {
	a.wg.Wait()
}
