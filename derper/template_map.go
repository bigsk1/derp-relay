package main

import (
	"encoding/json"
	"errors"
	"flag"
	"net"
	"net/http"
	"regexp"
	"strings"

	"tailscale.com/tailcfg"
)

var (
	publicHTTPSPort = flag.Int("public-https-port", 443, "external TCP port to advertise in the relay map")
	hostnameLabel   = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
)

func registerTemplateMap(mux *http.ServeMux, host string, httpsPort, stunPort int, stun, tls bool) error {
	if !tls {
		return errors.New("the relay template requires TLS")
	}
	if !validTemplateHost(host) {
		return errors.New("DERP_HOST must be a fully qualified DNS name without a scheme, port, or trailing dot")
	}
	if httpsPort < 1 || httpsPort > 65535 {
		return errors.New("DERP_HTTPS_PORT must be between 1 and 65535")
	}
	if !stun {
		stunPort = -1
	}

	const regionID = 900
	relayMap := tailcfg.DERPMap{
		OmitDefaultRegions: true,
		Regions: map[tailcfg.DERPRegionID]*tailcfg.DERPRegion{
			regionID: {
				RegionID:   regionID,
				RegionCode: "self",
				RegionName: "Self-hosted relay",
				Nodes: []*tailcfg.DERPNode{{
					Name:     "self-1",
					RegionID: regionID,
					HostName: host,
					DERPPort: httpsPort,
					STUNPort: stunPort,
				}},
			},
		},
	}
	body, err := json.Marshal(relayMap)
	if err != nil {
		return err
	}
	mux.HandleFunc("GET /derpmap.json", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			http.Error(w, "Fetch the relay map over HTTPS.", http.StatusUpgradeRequired)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body)
	})
	return nil
}

func validTemplateHost(host string) bool {
	if len(host) > 253 || net.ParseIP(host) != nil {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if !hostnameLabel.MatchString(label) {
			return false
		}
	}
	return true
}
