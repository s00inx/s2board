package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/s00inx/stdesk/internal/network"
	"github.com/s00inx/stdesk/internal/storage"
)

// TODO: инкапсулировать узел и сделать его методами все функции !!

func dlhandler(w http.ResponseWriter, r *http.Request) {
	hval := r.PathValue("hash")

	path := filepath.Join("data/blobs/" + hval[:2] + "/" + hval)
	w.Header().Set("Content-Disposition", "attachment; filename=\"download\"")

	http.ServeFile(w, r, path)
}

func InitNetwork() {
	st := storage.Storage{
		BaseDir: "data/",
		KeyDir:  "data/node.key",
	}
	curnode, _ := network.NodeConnect(st.KeyDir)
	curnode.Storage = st

	liface, ipstr := network.GetLocalIface()
	if liface == nil {
		fmt.Println("error configuring your web interface/")
	}

	go curnode.Discover(context.Background())

	port := "8080" // запускаем пока на порту 8080
	url := fmt.Sprintf("http://%s:%s", ipstr, port)

	fmt.Println("setup...")
	fmt.Println("mdns addr: http://stshare.local:8080")
	fmt.Printf("reserve addr: %s\n", url)
	network.PrintQr(url)

	mdnsrv, err := network.InitMdns(liface, curnode.UID)
	if err != nil {
		fmt.Println("mDns conf error: ", err)
	}
	defer mdnsrv.Shutdown()

	mux := http.NewServeMux()

	// для теста
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		os.Stdout.Write([]byte("new req\n"))

		finaljson := curnode.ProcessFile("test.txt", "test", "blAnk")

		//temp
		type Hash struct {
			Hash string `json:"filehash"`
		}
		fhash := Hash{}
		json.Unmarshal(finaljson, &fhash)

		w.Write([]byte("Node ID: " + curnode.UID + "\n\n\n" +
			string(finaljson) + "\n" + url + "/api/dl/" + fhash.Hash))
	})
	mux.HandleFunc("GET /", h)
	mux.HandleFunc("GET /api/dl/{hash}", dlhandler)

	log.Fatal(http.ListenAndServe(":"+port, mux))
}
