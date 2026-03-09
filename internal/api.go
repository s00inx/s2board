package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func InitNetwork() {
	curnode, _ := NodeConnect("node.key")

	liface, ipstr := GetLocalIface()
	if liface == nil {
		fmt.Println("error configuring your web interface/")
	}

	go discoverConns(context.Background()) // !! ДО запуска

	port := "8080" // запускаем пока на порту 8080
	url := fmt.Sprintf("http://%s:%s", ipstr, port)

	fmt.Println("setup...")
	fmt.Println("mdns addr: http://stshare.local:8080")
	fmt.Printf("reserve addr: %s\n", url)
	PrintQr(url)

	mdnsrv, err := InitMdns(liface, curnode.UID)
	if err != nil {
		fmt.Println("mDns conf error: ", err)
	}
	defer mdnsrv.Shutdown()

	// для теста
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Node ID: " + curnode.UID))
	})

	log.Fatal(http.ListenAndServe(":"+port, h))
}
