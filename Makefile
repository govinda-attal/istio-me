.PHONY: compile_proto generate_gw generate_swagger



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
	mkdir dist
	mkdir dist/config
	mkdir dist/api
	GOOS=linux GOARCH=amd64 go build -o ./dist/$(APP_NAME) .
	cp ./test/fixtures/app-config-local.yaml ./dist/config/app-config.yaml
	cp ./api/*.json ./dist/api/

local-providers-start:
	docker-compose up db adminer redis swagger-ui


local-serve: build
	cd dist && ./$(APP_NAME) serve

clean:
	rm ./dist/ -rf

pack:
	docker build --build-arg APP_NAME=$(APP_NAME) -t gattal/$(APP_NAME):$(TAG) .

upload:
	docker push gattal/$(APP_NAME):$(TAG)	

run:
	docker run --name cabride-api -d -v ./test/fixtures:/app/config -p $(HOST_PORT):9080 gattal/$(APP_NAME):$(TAG) sh -c "sleep 5s && ./cabride serve --config ./config/app-config-local.yaml"

ship: init test pack upload clean	