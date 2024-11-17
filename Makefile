vet:
	go vet ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

test:
	go test -race -cover -coverprofile=coverage.txt ./...