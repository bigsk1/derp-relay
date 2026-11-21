# Self-hosted DERP relay

Run a DERP relay on your own VPS and use it with WhatsUp. The container runs upstream Tailscale `derper` and serves a relay map for your hostname at `/derpmap.json`.

## Quick start on a dedicated VPS

Use a Linux VPS with Docker and Docker Compose. Point a DNS **A** record for your relay hostname at the VPS. If you also publish an AAAA record, make sure IPv6 reaches the same relay. Keep the relay hostname **DNS only** with Cloudflare or similar services. DERP needs a direct connection and STUN uses UDP.

| Inbound port | Purpose |
| --- | --- |
| TCP 80 | Let's Encrypt HTTP validation |
| TCP 443 | DERP over TLS and the WhatsUp relay map |
| UDP 3478 | STUN for discovering direct paths |

Allow these ports in the VPS provider firewall. Docker publishes them directly, so check the provider firewall even if you also use UFW. Keep SSH and any management dashboard limited to the access you intend.

From a checkout of this repository, copy the example and set `DERP_HOST` in `.env`. Use the bare hostname, such as `relay.example.com`.

```sh
cp .env.example .env
# Edit .env and set DERP_HOST before continuing.
docker compose -f docker-compose.yml -f compose.build.yml up -d --build
```

This builds from the checkout. If you use a published registry image, run `docker compose up -d` with the base file alone. It defaults to the first release image, `ghcr.io/bigsk1/derp-relay:v0.1.0`. Set `DERP_IMAGE` to use another reviewed release or digest. A fresh checkout can always use the source build above.

The relay uses Let's Encrypt and keeps its key and certificate cache in a named volume. The first HTTPS request may take a moment while the certificate is issued.

```sh
curl --fail --silent --show-error https://relay.example.com/derpmap.json
curl --fail --silent --show-error https://relay.example.com/derp/probe
```

Paste the **map URL**, including `/derpmap.json`, into WhatsUp's relay field when creating or joining an invitation. Use the same relay on both devices and confirm the expected connection behavior. For a fallback check, use two devices on networks where a direct path cannot form, confirm WhatsUp shows **Relayed**, then exchange a message. A successful map request verifies HTTPS and configuration delivery. The relayed message checks the packet path. [WhatsUp connection guide](https://github.com/bigsk1/whatsup#connections-and-relays)

## Deploy from a dashboard

The registry image must be published and readable for an image-based import. Use a deployment server with the ports above available or follow the shared-server setup below.

| Dashboard | Setup |
| --- | --- |
| Coolify | Add a Docker Compose resource from the Git repository and select `docker-compose.yml`. Set `DERP_HOST`, use Raw Compose, and leave the service's proxy Domains field empty. |
| Dokploy | Create a Compose service with the Git source and `docker-compose.yml`. Set `DERP_HOST` in Environment, choose Docker Compose mode, and leave the service's proxy domain unset. |
| Hostinger | Open Docker Manager and choose **Compose from URL**. Import the raw `docker-compose.yml` URL from a release of this repo, set `DERP_HOST` in the project configuration, review the port mappings, and deploy. |

Keep the published ports in the Compose file. The dashboard manages the container while clients connect directly to the VPS. [Coolify Compose](https://coolify.io/docs/docs/applications/builds/docker-compose) · [Dokploy Compose](https://docs.dokploy.com/docs/core/docker-compose) · [Hostinger Docker Manager](https://www.hostinger.com/support/12040815-how-to-deploy-your-first-container-with-hostinger-docker-manager/)

## Share a VPS that already has a web proxy

Coolify, Dokploy, and other proxies often use TCP 80 and 443. The manual TLS override publishes TCP **8443** and UDP **3478**, leaving those proxy ports free. It requires Docker Compose **2.24.4 or newer** and a trusted certificate you maintain outside the relay. [Compose override syntax](https://docs.docker.com/reference/compose-file/merge/#replace-value)

1. Obtain a certificate for the relay hostname through your normal certificate provider. A DNS challenge can issue one while another service owns TCP 80 and 443.
2. On the VPS, put the full certificate chain in `relay.example.com.crt` and the private key in `relay.example.com.key` in a dedicated directory. Both filenames must match `DERP_HOST`. Allow the container's numeric user or group **10001** to read those files and traverse the directory. Keep the key restricted to that service and the operator.
3. Set `DERP_CERT_DIR` to that absolute directory in `.env`. If you want a different HTTPS port, set `DERP_HTTPS_PORT`. Open the chosen TCP port and UDP 3478 in the provider firewall.

```sh
docker compose -f docker-compose.yml -f compose.manual-tls.yml up -d
```

Add `-f compose.build.yml` before `-f compose.manual-tls.yml` to build locally. On a dashboard with a single YAML editor, render the combined registry configuration on the target Linux host, then import the result.

```sh
docker compose -f docker-compose.yml -f compose.manual-tls.yml config
```

Use `https://relay.example.com:8443/derpmap.json` with the default shared-server port. The map advertises the same port to WhatsUp. Restart the relay after renewing the mounted certificate.

## Privacy and maintenance

- The relay admits compatible clients that can reach it. WhatsUp still authenticates contacts and encrypts their traffic. A relay operator can see client IP addresses and traffic timing and volume.
- The container runs as a non-root user with a read-only root filesystem. The state volume stores the relay key and, in automatic TLS mode, certificate data. Treat it as private and preserve it across upgrades.
- The default limits are 25 accepted TCP connections per second with a burst of 50 on the DERP listener, and 2 MiB/s inbound per DERP client with a 4 MiB burst. These limits apply to resource use. The relay can still consume the VPS transfer allowance. UDP STUN stays separate.
- The template adds no request access logging or telemetry. Docker keeps up to three 5 MiB log files per container. Docker rotates these by size. You can check errors locally with `docker compose logs --tail 40 relay`.
- The container health check checks the local TCP listener. Verify public TLS, the map URL, and a relayed connection from outside the VPS before sharing the endpoint.
- When upgrading, select a reviewed image tag or digest, pull it, and recreate the service with the same Compose files and state volume. Keep the previous version handy in case you need to roll back.

[Upstream DERP deployment and maintenance guidance](https://github.com/tailscale/tailscale/blob/v1.104.7/cmd/derper/README.md)

## Build and publish the image

`docker build -t derp-relay:local .` runs the relay-map tests and creates an image for your local Docker architecture. The Dockerfile records the base image digests and verifies the upstream Go module checksums.

Dashboard imports can use a public image from GHCR or another registry. To publish the first image from a signed-in maintainer environment, use a Buildx builder that supports multi-platform builds.

```sh
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/bigsk1/derp-relay:v0.1.0 --push .
```

Make the registry package public before sharing the Compose URL. [Docker multi-platform builds](https://docs.docker.com/build/building/multi-platform/) · [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
