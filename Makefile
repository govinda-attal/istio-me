.PHONY: compile_proto generate_gw generate_swagger

include .env
export $(shell sed 's/=.*//' .env)

TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)

compile_proto:
	cd api/; protoc -I. -I=$(GOPATH)/src/github.com/protocolbuffers/protobuf/src \
        -I=$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I$(GOPATH)/src  --go_out=plugins=grpc:$(GOPATH)/src/  *.proto	

	
generate_gw:
	cd api/; protoc -I/usr/local/include -I. \
		-I$(GOPATH)/src \
		-I=$(GOPATH)/src/github.com/protocolbuffers/protobuf/src \
        -I=$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:$(GOPATH)/src/ *.proto


generate_swagger:
	cd api/; protoc -I/usr/local/include -I. \
		-I$(GOPATH)/src \
		-I=$(GOPATH)/src/github.com/protocolbuffers/protobuf/src \
        -I=$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  		--swagger_out=logtostderr=true:. *.proto

init:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/onsi/gomega/...
	go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
	go get -u google.golang.org/grpc
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

install: init
	rm -rf ./vendor
	dep ensure

test: install
	ginkgo -r

build: 
	rm -rf ./dist
	mkdir -p dist/config
	mkdir -p dist/api
	GOOS=linux GOARCH=amd64 go build -o ./dist/$(APP_NAME) .
	cp ./test/fixtures/config.yaml ./dist/config.yaml
	cp ./api/*.json ./dist/api/


serve: build
	cd dist && ./$(APP_NAME) greet

clean:
	rm ./dist/ -rf

pack:
	docker build -t gattal/$(APP_NAME):$(TAG) .
	docker tag gattal/$(APP_NAME):$(TAG) gattal/$(APP_NAME):latest

upload:
	docker push gattal/$(APP_NAME):$(TAG)
	docker push gattal/$(APP_NAME):latest	

run:
	docker run --name istio-me -d -v=$(CURDIR)/test/fixtures/config.yaml:/app/config.yaml  -p $(HOST_PORT):9080 gattal/$(APP_NAME):$(TAG) "greet"

ship: init test pack upload clean	