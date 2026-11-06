# Build from the repository root: docker build -f docker/Dockerfile -t re-psn .  (docker/README.md)
# Both bases are pinned by digest (the multi-arch index). Dependabot's docker ecosystem
# updates the tag and the digest together; do not edit the digest by hand.
FROM golang:1.26-alpine3.24@sha256:df23b876380eb556f22bbba2fa57493e8684fc4fcabf06399fbfb65359978d28 AS build
WORKDIR /src
COPY go.mod .
COPY server/ server/
RUN cd server && CGO_ENABLED=0 go build -o /re-psn .

FROM alpine:3.24@sha256:3f4d6b32e26703c6082839234f43769d5046b3ac7c910033911d1e71d8cd630a
# A fixed, unprefixed uid/gid so a compose file can say `user: "10001:10001"`. No home, no shell.
RUN addgroup -S -g 10001 re-psn && adduser -S -D -H -u 10001 -G re-psn -s /sbin/nologin re-psn
COPY --from=build /re-psn /usr/local/bin/re-psn
USER 10001:10001
EXPOSE 36100/tcp 36100/udp 36101/tcp
# Settings come from RE_PSN_* variables (the old PSNR_* names still work;
# docs/running.md). The status page is off unless RE_PSN_HTTP is set
# (36101 is its usual port); off localhost, set RE_PSN_HTTP_TOKEN too.
ENTRYPOINT ["re-psn"]
