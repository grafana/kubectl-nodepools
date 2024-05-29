PKG ?= ./...

checks:
	go vet $(PKG)
	go run honnef.co/go/tools/cmd/staticcheck@latest $(PKG)
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run $(PKG)

test:
	go test $(PKG)

benchmark:
	go test -run='^$$' -bench=. -benchmem
