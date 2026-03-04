.PHONY: install vet

install:
	go install honnef.co/go/tools/cmd/staticcheck@latest

vet:
	gofmt -w .
	go vet ./...
	staticcheck ./...
