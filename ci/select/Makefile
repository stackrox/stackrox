BIN := bin/ci-select

.PHONY: all
all: build

.PHONY: build
build:
	go build -trimpath -o $(BIN) .

.PHONY: clean
clean:
	rm -rf bin
