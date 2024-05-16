PROTOC_ARGS_API=--go_out=proto --go_opt=paths=source_relative --go-grpc_out=proto --go-grpc_opt=paths=source_relative
PROTOC_ARGS_TESTS=--go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

# Absolute path to directory of this Makefile
ROOT_DIR=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

install_golang_proto_compiler:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

proto:
	protoc $(PROTOC_ARGS_API) --proto_path=proto proto/*.proto

install_mockgen:
	go install go.uber.org/mock/mockgen@latest

generate_mock_files:
	mockgen -source internal/control_plane/control_plane/persistence/interface.go > mock/mock_persistence/mock_persistence.go
	mockgen -source internal/control_plane/control_plane/core/interface.go > mock/mock_core/mock_core.go
	mockgen -source proto/control_plane_interface_grpc.pb.go -package mock_cp_api > mock/mock_cp_api/mock_cp_api.go

build_rootfs:
	sudo rm -rf $(ROOT_DIR)/configs/firecracker/app $(ROOT_DIR)/configs/firecracker/rootfs.ext4
	sudo $(ROOT_DIR)/configs/firecracker/image.sh \
		-d $(ROOT_DIR)/configs/firecracker/ \
		-s $(ROOT_DIR)/workload \
		-r $(ROOT_DIR)/configs/firecracker/rootfs.ext4

empty_container:
	docker build \
		-f Dockerfile \
		-t cvetkovic/dirigent_empty_function .
	docker push cvetkovic/dirigent_empty_function:latest

trace_container:
	docker build \
		-f Dockerfile \
		-t cvetkovic/dirigent_trace_function .
	docker push cvetkovic/dirigent_trace_function:latest
