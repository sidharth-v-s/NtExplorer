.PHONY: run build test validate fmt vet check dist-windows clean

BINARY := ntexplorer
VERSION := 0.4.0

run:
	go run ./cmd/ntexplorer

build:
	go build -trimpath -ldflags "-s -w" -o dist/$(BINARY) ./cmd/ntexplorer

test:
	go test ./...

validate:
	go run ./scripts/validate-data.go

fmt:
	gofmt -w .

vet:
	go vet ./...

check: fmt vet test validate

dist-windows:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/$(BINARY).exe ./cmd/ntexplorer
	@echo "Built dist/$(BINARY).exe ($(VERSION))"

clean:
	rm -rf dist exports
