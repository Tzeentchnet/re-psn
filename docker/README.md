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

## Podman

The same Dockerfile builds and runs with Podman, rootless included (ports above 1024 need no root):

```
podman build -f docker/Dockerfile -t psnr .

podman run -d --name psnr --restart=on-failure \
  -p 36100:36100/tcp -p 36100:36100/udp \
  psnr
```

Add `-e PSNR_<FLAG>=value` for any setting, or `--env-file psnr.env` (copy `deploy/psnr.env.example` from the release archive's `examples/`, or from
this repository, and keep `psnr.env` private). Status page, from this machine only:

```
-e PSNR_HTTP=0.0.0.0:36101 -e PSNR_HTTP_TOKEN=<a long random string> -p 127.0.0.1:36101:36101
```

`podman logs psnr` shows the log; `podman rm -f psnr` stops and removes it. Tested with Podman 6.1 on Linux: the image builds, a TCP connection to the
published port is accepted, the status page answers 401 without the token and 200 with it, and the restart policy is set. Not tested: UDP through the
published port, compose (`podman compose` needs a compose provider), a systemd unit for the container (Quadlet), and Windows or macOS Podman machines.

**Check the address psnr sees.** Rootless Podman forwards published ports itself, and in the test a connection from 127.0.0.1 was logged inside the
container as the machine's own LAN address. psnr uses the player's address for `-max-conns-per-addr`, the same-name reconnect rule and the address it
tells other players. Connect from another machine, look at the `hello` line in `podman logs psnr`, and make sure it shows that machine's address and not
the host's; if every player shows the host's address, those features lump everyone together. Running with `--network=host` (not tested here) avoids the
forwarding.

## Settings

The image has no fixed flags. Every flag is a `PSNR_<FLAG>` environment variable (`docs/running.md`, "Environment variables"); the compose
file shows `PSNR_ADDR`. If you change a port, change the published port and your firewall too.

The status page lists every online ID, so it is not reachable from the host by default. To reach it, bind it inside the container
and set a token, then publish the port on this machine only (the lines are in the compose file, commented):

```
PSNR_HTTP=0.0.0.0:36101
PSNR_HTTP_TOKEN=<a long random string>
```

Make the token with a random generator, not by hand. Any of these works (hex and base32 contain only characters that are safe in a `?token=` URL;
base64 can contain `+`, `/` and `=`, which then need URL-encoding):

```
openssl rand -hex 32                          # 64 hex characters (256 bits)
head -c 32 /dev/urandom | base32 | tr -d '='  # 52 base32 characters (256 bits), coreutils
openssl rand -base64 32                       # 44 characters, if you send it only as a Bearer header
```

All three were run (OpenSSL 3.5.9 and GNU coreutils) and gave the lengths shown.

A token set through the environment is visible to `docker inspect` and to other users on the host.

## Behind a reverse proxy

Set `PSNR_TRUSTED_PROXIES` to your proxy's address as the container sees it and `PSNR_PROXY_PROTOCOL=true`, and have the proxy send PROXY
protocol on the game TCP port. Publish UDP straight to the host. The release archives carry proxy examples in `examples/`; see
`docs/running.md`, "Behind a reverse proxy".

## The lab

`lab/run.sh` builds this image and `lab/Dockerfile` (both from the repository root) and runs the NAT scenarios.
