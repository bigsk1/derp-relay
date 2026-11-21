#!/bin/sh
set -eu
umask 077

: "${DERP_HOST:?Set DERP_HOST to the relay's public hostname}"
public_port=${DERP_HTTPS_PORT:-443}

case ${DERP_CERT_MODE:-letsencrypt} in
    letsencrypt)
        if [ "$public_port" != 443 ]; then
            echo "Use the manual TLS configuration for a published port other than 443." >&2
            exit 1
        fi
        cert_dir=/var/lib/derper/certs
        http_port=80
        ;;
    manual)
        cert_dir=/run/derp-tls
        http_port=-1
        ;;
    *)
        echo "DERP_CERT_MODE must be letsencrypt or manual." >&2
        exit 1
        ;;
esac

# Upstream creates the server key here on first boot.
exec /usr/local/bin/derper \
    -a :443 \
    -hostname "$DERP_HOST" \
    -public-https-port "$public_port" \
    -http-port "$http_port" \
    -certmode "${DERP_CERT_MODE:-letsencrypt}" \
    -certdir "$cert_dir" \
    -c /var/lib/derper/server.json \
    -home blank \
    -stun=true \
    -stun-port 3478 \
    -accept-connection-limit 25 \
    -accept-connection-burst 50 \
    -rate-config /etc/derper-rate.json
