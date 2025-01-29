.PHONY: all
all: clean build

.PHONY: build
build:
	go build -o bin/oslog-collector cmd/oslog-collector/main.go

.PHONY: clean
clean:
	go clean
