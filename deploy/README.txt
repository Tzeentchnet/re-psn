psnr deployment examples (text files; nothing here runs by itself)

psnr.env            every setting, commented, with its default. Copy it and edit.
psnr.service        a systemd unit (Linux).
traefik/            Traefik, TCP router with PROXY protocol v2. Tested with Traefik 3.1 under Podman.
nginx/psnr.conf     nginx stream with PROXY protocol. Not run.
haproxy/haproxy.cfg HAProxy with send-proxy-v2. Not run.

Behind a proxy you need, on psnr: PSNR_TRUSTED_PROXIES (only your own proxies; 0.0.0.0/0 trusts everyone) and
PSNR_PROXY_PROTOCOL=true. UDP is not proxied: publish it straight to the host. Ports in these files are the
defaults; if you change one, change the firewall too. The full text is docs/running.md in the repository.
