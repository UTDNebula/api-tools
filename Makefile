
EXEC_NAME?=api-tools

setup:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/cmd/goimports@latest

check: 
	go mod tidy
	go vet ./... 
	staticcheck ./...
	gofmt -w ./..
	goimports -w ./..

build: ./main/main.go
	go build -o $(EXEC_NAME) ./main/main.go

clean: $(EXEC_NAME)
	rm $(EXEC_NAME)
