# export GO111MODULE=on

# install
.PHONY: install
install:
	go get \
		github.com/golang/protobuf/protoc-gen-go \
		github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger \
		github.com/mwitkow/go-proto-validators/protoc-gen-govalidators \
		github.com/rakyll/statik

# gen cert
.PHONY: gen-cert
gen-cert:
	cd ./cert; sh gen.sh; cd ../

# gen stubs
.PHONY: gen
gen:
	@echo "====Gen stubs===="
	sh ./script/gen-proto.sh

# gen openapi
.PHONY: gen-openapi
gen-openapi:
	@echo "====Gen OpenAPI===="
	sh ./script/gen-openapi.sh

.PHONY: build
build:
	go build -o tbb ./cmd

#
.PHONY: run
run: lint clean
	@echo "====Run services===="
	docker-compose up -d --build
	sleep 1
	docker-compose exec app sh

.PHONY: log
log:
	docker-compose logs -f

# gofmt
.PHONY: fmt
fmt:
	# go fmt -mod=mod $(go list ./...)
	gofmt -l -s -w .

# go-lint
.PHONY: lint
lint: fmt
	golangci-lint run ./...

# go-test
.PHONY: test
test: lint
	go test -v ./...

# cleaning
.PHONY: clean
clean:
	@echo "====Cleaning env==="
	docker-compose down -v --remove-orphans

# grpc ui
# https://github.com/ktr0731/evans
.PHONY: cli
cli:
	evans -r repl -p 9090
