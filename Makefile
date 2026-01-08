
all: manga-mirror

clean:
	rm -rf $$HOME/go/bin/manga-mirror

manga-mirror: main.go utils/*.go
	gofmt -w .
	go build -race -o $$HOME/go/bin/manga-mirror main.go

deps:
	go mod tidy