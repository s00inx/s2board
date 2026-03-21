curlm:
	curl http://arch.local:8080

buildw:
	GOOS=WINDOWS GOARCH=amd64 go build -o stdesk.exe cmd/main.go

build:
	GOOS=LINUX GOARCH=amd64 go build cmd/main.go