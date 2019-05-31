# build stage
FROM golang:1.12-alpine AS build-env
RUN apk add --no-cache curl bash git openssh
ADD . /app
WORKDIR /app
RUN rm -rf ./dist && \
	mkdir -p dist/config && \
	mkdir -p dist/api

RUN GOOS=linux GOARCH=amd64 go build -o ./dist/istio-me . && \
	cp ./test/fixtures/config.yaml ./dist/config.yaml && \
	cp ./api/*.json ./dist/api/

# final stage
FROM alpine
RUN apk -U add ca-certificates
COPY --from=build-env /app/dist/* /app/
WORKDIR /app
RUN ls
ENTRYPOINT ["./istio-me"]