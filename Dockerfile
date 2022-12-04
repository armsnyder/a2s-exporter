# syntax=docker/dockerfile:1.4
FROM golang:1.19 AS builder
WORKDIR /build
COPY . .
ARG BUILD_VERSION=development
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/root/.cache/go-build go build -ldflags="-w -s -X 'main.buildVersion=$BUILD_VERSION'" -o /bin/a2s-exporter .

FROM scratch
COPY --from=builder /bin/a2s-exporter /bin/a2s-exporter
ENTRYPOINT ["/bin/a2s-exporter"]
