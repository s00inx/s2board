// инициализация мднс и локального адреса
package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/skip2/go-qrcode"
)

// !!: mdns требует настройки avahi на линукс (ну либо можно просто выключить avahi-daemon.service))
// на windows 10+ и macOS запустится нативно потому что они поддерживают эту технологию из коробки

// инициализируем zeroconf-сервис http://stdesk.local:8080, и передаем туда наш найденный сетевой интерфейс
// и uid для идентификации ноды
func InitMdns(ip *net.Interface, uid string) (*zeroconf.Server, error) {
	serv, err := zeroconf.Register(
		"stdesk",
		"_http._tcp",
		"local.",
		8080,
		[]string{
			"txtv=1",
			"mode=active",
			"uid=" + uid,
		},
		[]net.Interface{*ip},
	)

	if err != nil {
		return nil, fmt.Errorf("mdns error: %w", err)
	}

	return serv, err
}

// найти сетевой интерфейс для айпи который смотрит в локальную сеть
func GetLocalIface() (*net.Interface, string) {
	// все сетевые интерфейсы приклепленные к сетевой карте
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, ""
	}

	// пеербираем интерфейсы (обычно loopback и wlp0s)
	for _, ie := range ifaces {
		addrs, _ := ie.Addrs() // адреса интерфейса

		for _, a := range addrs {
			// проверка поднят ли вообще интефейс, не будем ли мы пытаться сделать все через выключенный wl например))
			if ie.Flags&net.FlagUp == 0 || ie.Flags&net.FlagLoopback != 0 {
				continue
			}

			// проверяем что это не мусор и не лупбек (lo)
			if ipmask, ok := a.(*net.IPNet); ok && !ipmask.IP.IsLoopback() {
				if ipmask.IP.To4() != nil {
					return &ie, ipmask.IP.String()
				}
			}
		}
	}

	// в худшем случае вернем нил в качестве интерфейса (потому что не нашли) и локалхост
	return nil, "127.0.0.1"
}

// печатает qr на осноснвой адрес (100% надежность прям в консоль)
func PrintQr(url string) error {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return err
	}

	fmt.Println(q.ToSmallString(false))
	return nil
}

// пир это просто ещё 1 нода в зоне видимости,
// мы храним активные ноды, чтобы обмениваться данными
type Peer struct {
	UID  string
	IP   string
	Port int

	LastSeen time.Time
}

var ActiveConns sync.Map

func discoverConns(ctx context.Context) {
	reslv, err := zeroconf.NewResolver(nil)
	if err != nil {
		panic(err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(res <-chan *zeroconf.ServiceEntry) {
		for entry := range res {
			var uid string
			for _, f := range entry.Text {
				if len(f) > 4 && f[:4] == "uid=" {
					uid = f[4:]
				}
			}

			if uid != "" {
				ActiveConns.Store(uid,
					Peer{
						UID:      uid,
						IP:       entry.AddrIPv4[0].String(),
						Port:     entry.Port,
						LastSeen: time.Now(),
					})
				fmt.Printf("\n[NEW PEER] Found node: %s at %s:%d\n", uid[:8], entry.AddrIPv4[0], entry.Port)
			}
		}
	}(entries)

	err = reslv.Browse(ctx, "_stdesk._tcp", "local.", entries)
	if err != nil {
		log.Println("resolver: failed to browse:", err)
	}

}
