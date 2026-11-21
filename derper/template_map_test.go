package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tailscale.com/tailcfg"
)

func TestTemplateMapAdvertisesConfiguredHostAndPorts(t *testing.T) {
	mux := http.NewServeMux()
	if err := registerTemplateMap(mux, "relay.example.test", 8443, 3478, true, true); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "https://relay.example.test/derpmap.json", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected response: code %d, headers %v", response.Code, response.Header())
	}
	var result tailcfg.DERPMap
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	region := result.Regions[900]
	if !result.OmitDefaultRegions || len(result.Regions) != 1 || region == nil || len(region.Nodes) != 1 {
		t.Fatalf("expected one self-hosted node in the map, got %+v", result)
	}
	node := region.Nodes[0]
	if node.RegionID != 900 || node.HostName != "relay.example.test" || node.DERPPort != 8443 ||
		node.STUNPort != 3478 || node.InsecureForTests {
		t.Fatalf("unexpected node: %+v", node)
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "http://relay.example.test/derpmap.json", nil))
	if response.Code != http.StatusUpgradeRequired {
		t.Fatalf("expected HTTPS requirement, got %d", response.Code)
	}
}

func TestTemplateMapRejectsInvalidConfiguration(t *testing.T) {
	for _, host := range []string{"", "relay", "https://relay.example.test", "relay.example.test.",
		"relay.example.test:443", "127.0.0.1", "relay..test", "-relay.example.test"} {
		if err := registerTemplateMap(http.NewServeMux(), host, 443, 3478, true, true); err == nil {
			t.Errorf("accepted invalid hostname %q", host)
		}
	}
	if err := registerTemplateMap(http.NewServeMux(), "relay.example.test", 8443, 3478, true, false); err == nil {
		t.Error("accepted a plaintext listener")
	}
	if err := registerTemplateMap(http.NewServeMux(), "relay.example.test", 65536, 3478, true, true); err == nil {
		t.Error("accepted an invalid public port")
	}
}
