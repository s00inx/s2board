TMPDIR := ./tmp
APP_NAME := s2board
MAIN_PATH := cmd/main.go

LDFLAGS := -s -w

.PHONY: build buildw clean test1 test2 test3

build:
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(APP_NAME) $(MAIN_PATH)

buildw:
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(APP_NAME).exe $(MAIN_PATH)

test1:
	go run $(MAIN_PATH) -port 8080 -dir $(TMPDIR)/data1 -name n8080

test2:
	go run $(MAIN_PATH) -port 8081 -dir $(TMPDIR)/data2 -name n8081

test3:
	go run $(MAIN_PATH) -port 8082 -dir $(TMPDIR)/data3 -name n8082

clean:
	rm -f $(APP_NAME) $(APP_NAME).exe
	rm -rf $(TMPDIR)/
	@echo "Cleaned up!"