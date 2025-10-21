PKG ?= ./...

checks:
	go vet $(PKG)
	go tool staticcheck $(PKG)
	go tool golangci-lint run $(PKG)

test:
	go test $(PKG)

benchmark:
	go test -run='^$$' -bench=. -benchmem
