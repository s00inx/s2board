// инициализация мднс и локального адреса
package internal

import (
	"fmt"
	"net"

	"github.com/grandcat/zeroconf"
	"github.com/skip2/go-qrcode"
)

// mdns требует настройки avahi на линукс (ну либо можно просто выключить avahi-daemon.service))
// на windows 10+ и macOS запустится нативно потому что они поддерживают эту технологию из коробки
func InitMdns(ip *net.Interface) (*zeroconf.Server, error) {
	serv, err := zeroconf.Register(
		"stdesk",
		"_http._tcp",
		"local.",
		8080,
		[]string{"txtv=1", "mode=active"},
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
