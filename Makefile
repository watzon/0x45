vet:
	go vet ./...

lint: vet
	golangci-lint run

test: lint vet
	go test -race -cover -coverprofile=coverage.txt ./...