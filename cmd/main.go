package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/s00inx/stdesk/internal"
)

func main() {
	liface, ipstr := internal.GetLocalIface()
	if liface == nil {
		fmt.Println("error configuring your web interface/")
	}

	port := "8080" // запускаем пока на порту 8080
	url := fmt.Sprintf("http://%s:%s", ipstr, port)

	fmt.Println("setup...")
	fmt.Println("mdns addr: http://stshare.local:8080")
	fmt.Printf("reserve addr: %s\n", url)
	internal.PrintQr(url)

	mdnsrv, err := internal.InitMdns(liface)
	if err != nil {
		fmt.Println("mDns conf error: ", err)
	}
	defer mdnsrv.Shutdown()

	// для теста
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	log.Fatal(http.ListenAndServe(":"+port, h))
}
