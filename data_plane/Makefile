PROTOC_ARGS_API=--go_out=api/proto --go_opt=paths=source_relative --go-grpc_out=api/proto --go-grpc_opt=paths=source_relative
PROTOC_ARGS_TESTS=--go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

install_golang_proto_compiler:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

proto:
	protoc $(PROTOC_ARGS_API) --proto_path=api/proto api/proto/*.proto
	protoc $(PROTOC_ARGS_TESTS) tests/proto/*.proto
