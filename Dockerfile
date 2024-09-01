FROM registry.cn-hangzhou.aliyuncs.com/jinli09051005/tools:golang-1.21 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY . .
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.io,direct
#RUN go mod tidy && go mod vendor && GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o manager main.go
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o manager main.go

FROM registry.cn-hangzhou.aliyuncs.com/jinli09051005/tools:busybox-latest
WORKDIR /app
COPY --from=builder /app/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
