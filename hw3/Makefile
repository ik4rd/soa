.PHONY: generate check integration

generate:
	mkdir -p bin
	GOBIN=$(CURDIR)/bin go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	GOBIN=$(CURDIR)/bin go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
	PATH="$(CURDIR)/bin:$$PATH" protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/flight/v1/flight.proto
	go run ./scripts/stripcomments api/flight/v1/flight.pb.go api/flight/v1/flight_grpc.pb.go

check:
	go test -race ./...
	go vet ./...

integration:
	docker compose --profile test run --build --rm tests
