# Running psnr in a container

Files here: `Dockerfile` (a two-stage build: Go in the first stage, a small Alpine image with one static binary in the second),
`docker-compose.yml`, and `Dockerfile.dockerignore` (keeps everything but `go.mod` and `server/` out of the build context).
No image is published yet; you build it from this repository.

## Build and run

From the repository root:

```
docker build -f docker/Dockerfile -t psnr .
docker run --rm -p 36100:36100 -p 36100:36100/udp psnr

docker compose -f docker/docker-compose.yml up -d      # the same, with the ports and a restart policy
```

The build context is the repository root, not this folder: the Dockerfile copies `go.mod` and `server/` from there. Podman works the same
way (`podman build -f docker/Dockerfile .`).

## Settings

The image has no fixed flags. Every flag is a `PSNR_<FLAG>` environment variable (`docs/running.md`, "Environment variables"); the compose
file shows `PSNR_ADDR`. If you change a port, change the published port and your firewall too.

The status page lists every online ID, so it is not reachable from the host by default. To reach it, bind it inside the container
and set a token, then publish the port on this machine only (the lines are in the compose file, commented):

```
PSNR_HTTP=0.0.0.0:36101
PSNR_HTTP_TOKEN=<a long random string>
```

A token set through the environment is visible to `docker inspect` and to other users on the host.

## Behind a reverse proxy

Set `PSNR_TRUSTED_PROXIES` to your proxy's address as the container sees it and `PSNR_PROXY_PROTOCOL=true`, and have the proxy send PROXY
protocol on the game TCP port. Publish UDP straight to the host. The release archives carry proxy examples in `examples/`; see
`docs/running.md`, "Behind a reverse proxy".

## The lab

`lab/run.sh` builds this image and `lab/Dockerfile` (both from the repository root) and runs the NAT scenarios.
