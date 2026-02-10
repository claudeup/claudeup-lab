.PHONY: build test clean install

build:
	go build -o claudeup-lab ./cmd/claudeup-lab

test:
	go test ./... -v

clean:
	rm -f claudeup-lab

install: build
	mkdir -p $(HOME)/.local/bin
	cp claudeup-lab $(HOME)/.local/bin/
