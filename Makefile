.PHONY: build clean release

build:
	go build -o mtop main.go

clean:
	rm -f mtop mtop-linux-amd64 mtop-linux-arm64

release:
	GOOS=linux GOARCH=amd64 go build -o mtop-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -o mtop-linux-arm64 main.go
	@echo "Build release completed:"
	@ls -lh mtop-linux-*
