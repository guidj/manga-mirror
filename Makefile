
all: manga-mirror

clean:
	rm -rf ${GOPATH}/bin/manga-mirror

manga-mirror: main.go utils/*.go
	gofmt -w .
	go build -race -o ${GOPATH}/bin/manga-mirror main.go
