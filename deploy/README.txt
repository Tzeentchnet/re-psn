psnr deployment examples (text files; nothing here runs by itself)

psnr.env.example    every setting, commented, with its default. Copy it to psnr.env and edit; keep psnr.env private.
psnr.service        a systemd unit (Linux).
traefik/            Traefik, TCP router with PROXY protocol v2. Tested with Traefik 3.1 under Podman.
nginx/psnr.conf     nginx stream with PROXY protocol v1. Not run.
haproxy/haproxy.cfg HAProxy with send-proxy-v2. Not run.

Behind a proxy you need, on psnr: PSNR_TRUSTED_PROXIES (only your own proxies; 0.0.0.0/0 trusts everyone) and
PSNR_PROXY_PROTOCOL=true. UDP is not proxied: publish it straight to the host. Ports in these files are the
defaults. In the proxy examples the proxy owns 36100 (TCP game) and 36101 (status page) and psnr listens behind it on
127.0.0.1:36200 and 36201 (PSNR_TCP_ADDR, PSNR_HTTP). If you change a port, change the firewall too. The full text is docs/running.md in the repository.
