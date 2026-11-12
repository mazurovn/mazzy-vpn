// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright © 2026 Nik m (@mazurovn). All rights reserved.

package checks

import (
	"testing"

	"github.com/mazurovn/mazzy-vpn/desktop/monitor"
)

func TestMissingAddressStaysUnknown(t *testing.T) {
	doc, err := Parse(`{"unknown":[{"check":"ipv4","reason":"ip-api did not return an address"}],"antigravity":{"service":"antigravity","state":"indeterminate","reason":"unrecognized HTTP 200"}}`)
	if err != nil {
		t.Fatal(err)
	}
	v := Apply(monitor.View{Handshake: monitor.Unknown, Path: monitor.Unknown}, doc)
	if v.EgressLight != monitor.Unknown || v.Clean != monitor.Unknown || v.Service != monitor.Unknown {
		t.Fatalf("%+v", v)
	}
	if len(v.Points) != 0 {
		t.Fatal("invented a sample")
	}
}

func TestEligibleProbeRecordsLatency(t *testing.T) {
	ms := int64(40)
	f := false
	doc := Doc{
		IPv4: "203.0.113.8", Country: "AT", Proxy: &f, Hosting: &f,
		Timezone: "Europe/Vienna", TimezoneCountry: "AT",
		Antigravity: Service{State: "eligible", HTTP: 401, Reason: "auth boundary reached", LatencyMS: &ms},
	}
	bound := true
	doc.DNSBound = &bound
	doc.DNSCountry = "AT"
	v := Apply(monitor.View{}, doc)
	if v.EgressLight != monitor.Green || v.Service != monitor.Green || v.Clean != monitor.Green {
		t.Fatalf("lights egress %s service %s clean %s (%s)", v.EgressLight, v.Service, v.Clean, v.CleanText)
	}
	if len(v.Points) != 1 || !v.Points[0].HasMS || v.Points[0].MS != 40 {
		t.Fatalf("points %+v", v.Points)
	}
}
