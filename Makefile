curlm:
	curl http://arch.local:8080

buildw:
	GOOS=WINDOWS GOARCH=amd64 go build -o stdesk.exe cmd/main.go

build:
	GOOS=LINUX GOARCH=amd64 go build cmd/main.go

test1:
	go run ./cmd/main.go -port 8080 -dir ./data1 -name etst1

test2:
	go run ./cmd/main.go -port 8081 -dir ./data2 -name test2

cleant:
	grep ./data* | xargs rm -rf