FROM ghcr.io/bunnymediaserver/proto-builder:v0.0.1 AS builder
WORKDIR /proto

# Copy our repo
COPY . ./

# Lib dependencies
ENV GAU_VERSION "v0.0.26"
RUN go get github.com/BRUHItsABunny/go-android-utils@$GAU_VERSION

# Setup general includes
ENV PROTO_INC "-I ./ \
  -I ../ \
  -I ../../ \
  -I $GOPATH/src \
  -I $GOPATH/pkg/mod \
  -I $GOPATH/pkg/mod/github.com/!b!r!u!h!its!a!bunny/go-android-utils@$GAU_VERSION"

ENV TARGETS "api"
ENV PROTOC_VT_DRPC "protoc ${PROTO_INC} --go_out=. --plugin protoc-gen-go=${GOPATH}/bin/protoc-gen-go --go-grpc_out=. --plugin protoc-gen-go-grpc=${GOPATH}/bin/protoc-gen-go-grpc --go-vtproto_out=. --plugin protoc-gen-go-vtproto=${GOPATH}/bin/protoc-gen-go-vtproto --go-vtproto_opt=features=marshal+unmarshal+size ./*.proto"

RUN for target in ${TARGETS}; do cd /proto/$target && ${PROTOC_VT_DRPC} && find . -name "*.go" -type f -exec cp {} /proto/$target \; && rm -r /proto/$target/github.com; done
