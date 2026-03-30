.PHONY: generate

generate:
	@echo "Generating gRPC and Protobuf code with API_OPAQUE..."
	protoc -I=proto \
		--go_out=. \
		--go_opt=module=github.com/dmnAlex/sberscribe \
		--go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/dmnAlex/sberscribe \
		proto/*.proto
	@echo "Done."