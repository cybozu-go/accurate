# Build the manager binary
FROM quay.io/cybozu/golang:1.16-focal as builder

COPY ./ .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o innu-controller ./cmd/innu-controller

# the controller image
FROM scratch
LABEL org.opencontainers.image.source https://github.com/cybozu-go/innu

COPY --from=builder /work/innu-controller ./
USER 10000:10000

ENTRYPOINT ["/innu-controller"]
