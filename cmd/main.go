package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/hashicorp/mdns"
)

func startMDNS(ip string) {
	hostName := "stdesk" // Имя, которое мы хотим

	service, err := mdns.NewMDNSService(
		hostName,           // Instance: "stdesk"
		"_http._tcp",       // Service type
		"local.",           // Domain
		hostName+".local.", // HostName: ОБЯЗАТЕЛЬНО "stdesk.local."
		8080,
		[]net.IP{net.ParseIP(ip)},
		[]string{"LocalHub"},
	)

	if err != nil {
		log.Printf("mdns error: %v", err)
		return
	}

	_, err = mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		log.Printf("mdns err: %v", err)
	}
}

// найти айпи который смторит в локальную сеть чтобы запустить наш сервер именно на нем
func getLocalIP() string {
	// все сетевые интерфейсы приклепленные к сетевой карте
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, a := range addrs {
		// убеждаемся чт это валидный адрес с помощью приведения к интерфейсу
		if ipmask, ok := a.(*net.IPNet); ok && !ipmask.IP.IsLoopback() {
			if ipmask.IP.To4() != nil {
				return ipmask.IP.String()
			}
		}
	}

	// в случае чего возвращаем локалхост
	return "127.0.0.1"
}

func main() {
	ip := getLocalIP()
	port := "8080" // запускаем пока на порту 8080

	fmt.Println("setup...")
	fmt.Println("mdns addr: http://stshare.local:8080")
	fmt.Printf("reserve addr: http://%s:%s\n", ip, port)

	startMDNS(ip)

	// для теста
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
		fmt.Println("new conn : ", r.URL.User.Username())
	})

	log.Fatal(http.ListenAndServe(":"+port, h))

}
