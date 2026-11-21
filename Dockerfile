FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS build

ARG TARGETOS
ARG TARGETARCH
ENV GOTOOLCHAIN=local

RUN --mount=type=cache,target=/go/pkg/mod <<'SH'
set -eu
go telemetry off
go mod download -json tailscale.com@v1.104.7 > /tmp/upstream.json
grep -F '"Sum": "h1:YqImcXyNIe25x7Qwei8PgbtkjFd3pKed9cFY0Flkitc="' /tmp/upstream.json
grep -F '"GoModSum": "h1:pNl682NgUoac2/Um3khv8pduX3D1YPBZTL067+zo8wY="' /tmp/upstream.json
mkdir /src
cp -R /go/pkg/mod/tailscale.com@v1.104.7/. /src/
chmod u+w /src/cmd/derper /src/cmd/derper/derper.go
SH

WORKDIR /src
COPY derper/ cmd/derper/
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build <<'SH'
set -eu
# Register a read-only map on the same TLS listener. Leave DERP and STUN to upstream.
file=cmd/derper/derper.go
hook='^[[:blank:]]*mux := http\.NewServeMux()$'
test "$(grep -c "$hook" "$file")" = 1
sed -i "/$hook/a\\	if err := registerTemplateMap(mux, *hostname, *publicHTTPSPort, *stunPort, *runSTUN, serveTLS); err != nil { log.Fatal(err) }" "$file"
gofmt -w "$file"
CGO_ENABLED=0 go test -mod=readonly -run '^TestTemplate' ./cmd/derper
install -d -m 0700 -o 10001 -g 10001 /out/state /out/state/certs
CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
  go build -mod=readonly -trimpath -ldflags='-s -w' -o /out/derper ./cmd/derper
test -s /etc/ssl/certs/ca-certificates.crt
SH

FROM docker.io/library/alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
LABEL org.opencontainers.image.source="https://github.com/bigsk1/derp-relay" \
      org.opencontainers.image.description="A self-hosted DERP relay with a WhatsUp-compatible relay map"
COPY --from=build /out/derper /usr/local/bin/derper
COPY --from=build /src/LICENSE /usr/share/licenses/tailscale/LICENSE
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chown=10001:10001 /out/state/ /var/lib/derper/
COPY --chmod=0555 entrypoint.sh /usr/local/bin/relay-entrypoint
COPY --chmod=0444 rate.json /etc/derper-rate.json

ENV HOME=/var/lib/derper
USER 10001:10001
EXPOSE 80/tcp 443/tcp 3478/udp
ENTRYPOINT ["/usr/local/bin/relay-entrypoint"]
