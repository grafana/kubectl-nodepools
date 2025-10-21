PKG ?= ./...

TOOL = go tool -modfile=go.tool.mod

checks:
	go vet $(PKG)
	$(TOOL) staticcheck $(PKG)
	$(TOOL) golangci-lint run $(PKG)

test:
	go test $(PKG)

benchmark:
	go test -run='^$$' -bench=. -benchmem
