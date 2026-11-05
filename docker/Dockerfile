# Build from the repository root: docker build -f docker/Dockerfile .  (docker/README.md)
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod .
COPY server/ server/
RUN cd server && CGO_ENABLED=0 go build -o /psnr .

FROM alpine:3.20
COPY --from=build /psnr /usr/local/bin/psnr
EXPOSE 36100/tcp 36100/udp 36101/tcp
# Settings come from PSNR_* variables (docs/running.md). The status page is off
# the network until you set PSNR_HTTP and PSNR_HTTP_TOKEN.
ENTRYPOINT ["psnr"]
