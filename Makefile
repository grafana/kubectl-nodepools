PKG ?= ./...

checks:
	go vet $(PKG)
	go tool -modfile=go.tool.mod staticcheck $(PKG)
	go tool -modfile=go.tool.mod golangci-lint run $(PKG)

test:
	go test $(PKG)

benchmark:
	go test -run='^$$' -bench=. -benchmem
