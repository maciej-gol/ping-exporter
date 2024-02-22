# Cross-compile the binary for amd64 and aarch64 armv7:
FROM golang:1.22.0 as builder

COPY . /go/src/github.com/maciej-gol/ping-exporter
WORKDIR /go/src/github.com/maciej-gol/ping-exporter
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/ping-exporter .

FROM scratch

COPY --from=builder /out/ping-exporter .
ENTRYPOINT ["./ping-exporter"]
