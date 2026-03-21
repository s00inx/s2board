// инициализация мднс и локального адреса
package network

import (
	"fmt"
	"net"
	"os"

	"github.com/grandcat/zeroconf"
	"github.com/skip2/go-qrcode"
)

// !!: mdns требует настройки avahi на линукс (ну либо можно просто sudo systemctl disable avahi-daemon.service для systemd))
// на windows 10+ и macOS запустится нативно потому что они поддерживают эту технологию из коробки

// инициализируем zeroconf-сервис http://stdesk.local:8080, и передаем туда наш найденный сетевой интерфейс
// и uid для идентификации ноды
func InitMdns(ip *net.Interface, uid string) (*zeroconf.Server, error) {
	if ip == nil {
		return nil, fmt.Errorf("can't find any valid net interface, please connect to hotspot")
	}

	hostname, _ := os.Hostname()

	// важно: эта функция под капотом использует dns-sd, то есть по сути устройство выходит в сеть под хостнеймом, но предоставляет услугу stdesk
	// (https://habr.com/ru/articles/839602/)
	serv, err := zeroconf.Register(
		hostname,
		"_stdesk._tcp",
		"local.",
		8080,
		[]string{
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

	// пеербираем интерфейсы (обычно loopback и wlan)
	for _, ie := range ifaces {
		addrs, _ := ie.Addrs() // адреса интерфейса

		for _, a := range addrs {
			// проверка поднят ли вообще интефейс, не будем ли мы пытаться сделать все через выключенный wlan например))
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
