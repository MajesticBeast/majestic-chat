gen-proto:
	@echo "Building serverside protobufs"
	@cd gochatserver && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	proto/gochat.proto
	@echo "Building clientside protobufs"
	@cd gochatclient && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	proto/gochat.proto

build:
	@echo "Building gochat server..."
	@cd gochatserver && go build -o gochat-server

	@echo "Building gochat client..."
	@cd gochatclient && go build -o gochat-client