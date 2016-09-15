
all: mangamirror

clean:
	rm -rf bin/

mangamirror: main.go utils/*.go
	mkdir -p bin
	go build -o bin/manga-mirror main.go
