PROTOC_ARGS=--go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

install_golang_proto_compiler:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

proto:
	protoc $(PROTOC_ARGS) api/proto/data_plane_interface.proto
	protoc $(PROTOC_ARGS) tests/proto/faas.proto
