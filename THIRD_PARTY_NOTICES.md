# Third-party notices

The relay image builds Tailscale's DERP server from [Tailscale v1.104.7](https://github.com/tailscale/tailscale/tree/v1.104.7). See the [upstream BSD 3-Clause license](https://github.com/tailscale/tailscale/blob/v1.104.7/LICENSE) and `/usr/share/licenses/tailscale/LICENSE` inside the image. The Dockerfile registers a small additional HTTPS endpoint for a relay map.

The image uses the [Alpine Linux Docker Official Image](https://hub.docker.com/_/alpine) and builds with the [Go Docker Official Image](https://hub.docker.com/_/golang).
