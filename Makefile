PROTO_DIR = proto
GO_OUT_DIR = proto/gen

.PHONY: install-tools
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.15.0
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.15.0

.PHONY: proto
proto:
	mkdir -p $(GO_OUT_DIR)
	protoc -I $(PROTO_DIR) \
		--go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(GO_OUT_DIR) --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=docs --openapiv2_opt=allow_merge=true,merge_file_name=swagger \
		$(PROTO_DIR)/*.proto

.PHONY: clean
clean:
	rm -rf $(GO_OUT_DIR)

.PHONY: all
all: install-tools proto