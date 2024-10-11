PKG ?= ./...

checks:
	go vet $(PKG)
	go run honnef.co/go/tools/cmd/staticcheck@v0.4.7 $(PKG)
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1 run $(PKG)

test:
	go test $(PKG)

benchmark:
	go test -run='^$$' -bench=. -benchmem
