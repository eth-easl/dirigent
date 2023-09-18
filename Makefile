PROTOC_ARGS_API=--go_out=api/proto --go_opt=paths=source_relative --go-grpc_out=api/proto --go-grpc_opt=paths=source_relative
PROTOC_ARGS_TESTS=--go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

install_golang_proto_compiler:
	go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

proto:
	protoc $(PROTOC_ARGS_API) --proto_path=api/proto api/proto/*.proto
	protoc $(PROTOC_ARGS_TESTS) tests/proto/*.proto

install_mockgen:
	go install go.uber.org/mock/mockgen@latest

generate_mock_files:
	mockgen -source internal/control_plane/persistence/interface.go > mock/mock_persistence/mock_persistence.go
	mockgen -source internal/control_plane/core/interface.go > mock/mock_core/mock_core.go
