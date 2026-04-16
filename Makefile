.PHONY: build test clean

build:
	go build -o bin/jotter .

test:
	go test ./...

clean:
	rm -rf bin/
