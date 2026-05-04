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
	// 1. Инициализация хранилища
	ist, est, err := storage.Init(a.cfg.DataDir)
	if err != nil {
		log.Fatalf("[FATAL] storage init error: %v", err)
	}
	a.internalst = ist
	a.extst = est

	// 2. Настройка P2P ноды
	node, err := network.ConnNode(filepath.Join(est.Dir, "node.key"), a.cfg.Port, a.cfg.Name)
	if err != nil {
		log.Fatalf("[FATAL] node connect error: %v", err)
	}

	node.Codec = codec.JSONCodec{}
	node.InternalStorage = ist
	node.FileStorage = est
	a.Node = node

	// Очистка и инициализация локальных данных
	a.Node.InternalStorage.Cleanvb()
	a.Node.InternalStorage.InitLocal()

	// 3. Сетевая настройка
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

	// 4. Подготовка HTTP сервера
	mux := a.setupRoutes()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.Port),
		Handler: mux,
	}

	// 5. Регистрация mDNS
	_, mdnsrv, err := network.InitMdns(liface, a.Node.UID, a.cfg.Name, a.cfg.Port)
	if err != nil {
		log.Println("[WARN] mDNS registration failed:", err)
	}

	// --- ЗАПУСК ГОРУТИН ---

	// Горутина: Discovery (поиск соседей)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		log.Println("[net] starting discovery...")
		a.Node.Discover(ctx)
	}()

	// Горутина: HTTP Server
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		log.Printf("[http] server listening on %s:%d", ipstr, a.cfg.Port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("[ERROR] http server error: %v", err)
		}
	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		<-ctx.Done()

		log.Println("[app] starting shutdown sequence...")

		if a.Node != nil {
			log.Println("[p2p] broadcasting Bye-packets to peers...")
			a.Node.Byew()
			time.Sleep(500 * time.Millisecond)
		}

		if mdnsrv != nil {
			mdnsrv.Shutdown()
		}

		// ШАГ 3: Гасим сервер
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)

		log.Println("[app] bye-logic finished safely")
	}()
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
