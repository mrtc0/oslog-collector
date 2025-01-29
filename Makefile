.PHONY: all
all: clean build

.PHONY: build
build:
	go build -o bin/oslog-collector .

.PHONY: clean
clean:
	go clean
