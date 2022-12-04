FROM golang:1.19 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY internal internal
COPY *.go .
ARG BUILD_VERSION=development
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s -X 'main.buildVersion=$BUILD_VERSION'" -o /bin/a2s-exporter .

FROM scratch
COPY --from=builder /bin/a2s-exporter /bin/a2s-exporter
ENTRYPOINT ["/bin/a2s-exporter"]
