all: build

build: deps
	go build -o fireFly .

deps:
	go mod tidy