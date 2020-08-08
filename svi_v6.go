// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"testing"
	"time"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/docker"
)

func sviNetV6Test(t *testing.T) {
	svi6OspfTest(t, "testdata/svi6/ospf/conf.yaml.tmpl")
}

func svi6OspfTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		svi6Carrier{docket},
		svi6Connectivity{docket},
		svi6OspfDaemons{docket},
		svi6OspfConfig{docket},
		svi6OspfNeighbors{docket},
		svi6OspfRoutes{docket},
		svi6OspfInterConnectivity{docket},
		svi6OspfFlap{docket},
		svi6Connectivity{docket},
		svi6OspfNeighbors{docket},
		svi6OspfRoutes{docket},
		svi6OspfInterConnectivity{docket},
		svi6OspfAdminDown{docket},
	)
}

type svi6Connectivity struct{ *docker.Docket }

func (svi6Connectivity) String() string { return "connectivity" }

func (svi6 svi6Connectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"H1", "2001:db8:1::1"},
		{"H2", "2001:db8:1::1"},
		{"R1", "2001:db8:1::2"},
		{"R1", "2001:db8:1::3"},
		{"R1", "2001:db8:2::2"},
		{"R2", "2001:db8:2::1"},
		{"R2", "2001:db8:3::3"},
		{"H3", "2001:db8:3::2"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(svi6.PingCmd(t, x.hostname, x.target))
	}
}

type svi6Carrier struct{ *docker.Docket }

func (svi6Carrier) String() string { return "carrier" }

func (svi6 svi6Carrier) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range svi6.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			assert.Comment("check carrier for", r.Hostname,
				"on", intf)
			assert.Nil(test.Carrier(r.Hostname, intf))
		}
	}
}

type svi6OspfDaemons struct{ *docker.Docket }

func (svi6OspfDaemons) String() string { return "daemons" }

func (svi6 svi6OspfDaemons) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range svi6.Routers {
		assert.Comment("ing FRR on", r.Hostname)
		out, err := svi6.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.Match(out, ".*ospf6d.*")
		assert.Match(out, ".*zebra.*")
	}
}

type svi6OspfConfig struct{ *docker.Docket }

func (svi6OspfConfig) String() string { return "config" }

func (svi6 svi6OspfConfig) Test(t *testing.T) {
	assert := test.Assert{t}
	assert.Comment("configuring OSPF v3")

	for _, r := range svi6.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := svi6.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "ipv6 forwarding")
			assert.Nil(err)
			if i.Upper != "" {
				continue
			}
			if r.Hostname != "H1" && r.Hostname != "H2" && intf != "br0" && intf != "dummy0" {
				_, err = svi6.ExecCmd(t, r.Hostname,
					"vtysh", "-c", "conf t", "-c", "interface "+intf, "-c", "ipv6 ospf6 network point-to-point")
				assert.Nil(err)
			}
			_, err = svi6.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "router ospf6", "-c", "interface "+intf+" area 0.0.0.0")
			assert.Nil(err)
			_, err = svi6.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "router ospf6", "-c", "redistribute connected")
			assert.Nil(err)
		}
	}
}

type svi6OspfNeighbors struct{ *docker.Docket }

func (svi6OspfNeighbors) String() string { return "neighbors" }

func (svi6 svi6OspfNeighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	timeout := 120

	for _, x := range []struct {
		hostname string
		peer     string
	}{
		{"H1", "0.0.0.1"},
		{"H2", "0.0.0.1"},
		{"R1", "1.0.0.1"},
		{"R1", "1.0.0.2"},
		{"R1", "0.0.0.2"},
		{"R2", "0.0.0.1"},
		{"R2", "1.0.0.3"},
		{"H3", "0.0.0.2"},
	} {
		found := false
		for i := timeout; i > 0; i-- {
			out, err := svi6.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show ipv6 ospf neighbor")
			assert.Nil(err)
			if !assert.MatchNonFatal(out, x.peer) {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No ospf neighbor found for %v", x.hostname)
		}
	}
}

type svi6OspfRoutes struct{ *docker.Docket }

func (svi6OspfRoutes) String() string { return "routes" }

func (svi6 svi6OspfRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	test.Pause.Prompt("stop")

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"H1", "2001:db8:2::/64"},
		{"H1", "2001:db8:3::/64"},
		{"H1", "2001:db8:0:192::2"},
		{"H1", "2001:db8:0:192::3"},
		{"H1", "2001:db8:1:192::1"},
		{"H1", "2001:db8:1:192::2"},

		{"H2", "2001:db8:2::/64"},
		{"H2", "2001:db8:3::/64"},
		{"H2", "2001:db8:0:192::1"},
		{"H2", "2001:db8:0:192::3"},
		{"H2", "2001:db8:1:192::1"},
		{"H2", "2001:db8:1:192::2"},

		{"R1", "2001:db8:3::/64"},
		{"R1", "2001:db8:0:192::1"},
		{"R1", "2001:db8:0:192::2"},
		{"R1", "2001:db8:0:192::3"},
		{"R1", "2001:db8:1:192::2"},
		{"R1", "2001:db8:3::/64"},

		{"R2", "2001:db8:1::/64"},
		{"R2", "2001:db8:0:192::1"},
		{"R2", "2001:db8:0:192::2"},
		{"R2", "2001:db8:0:192::3"},
		{"R2", "2001:db8:1:192::1"},

		{"H3", "2001:db8:1::/64"},
		{"H3", "2001:db8:2::/64"},
		{"H3", "2001:db8:0:192::1"},
		{"H3", "2001:db8:0:192::2"},
		{"H3", "2001:db8:1:192::1"},
		{"H3", "2001:db8:1:192::2"},
	} {
		found := false
		timeout := 60
		for i := timeout; i > 0; i-- {
			out, err := svi6.ExecCmd(t, x.hostname,
				"ip", "-6", "route", "show", x.route)
			assert.Nil(err)
			if !assert.MatchNonFatal(out, x.route) {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No ospf route for %v: %v", x.hostname, x.route)
		}
	}
}

type svi6OspfInterConnectivity struct{ *docker.Docket }

func (svi6OspfInterConnectivity) String() string { return "inter-connectivity" }

func (svi6 svi6OspfInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"H1", "2001:db8:1::1"},
		{"H1", "2001:db8:1::2"},
		{"H1", "2001:db8:2::1"},
		{"H1", "2001:db8:3::2"},
		{"H1", "2001:db8:3::3"},

		{"H2", "2001:db8:1::1"},
		{"H2", "2001:db8:1::3"},
		{"H2", "2001:db8:2::1"},
		{"H2", "2001:db8:3::2"},
		{"H2", "2001:db8:3::3"},

		{"R1", "2001:db8:1::2"},
		{"R1", "2001:db8:1::3"},
		{"R1", "2001:db8:2::2"},
		{"R1", "2001:db8:3::2"},
		{"R1", "2001:db8:3::3"},

		{"R2", "2001:db8:1::1"},
		{"R2", "2001:db8:1::2"},
		{"R2", "2001:db8:1::3"},
		{"R2", "2001:db8:2::1"},
		{"R2", "2001:db8:3::3"},

		{"H3", "2001:db8:1::1"},
		{"H3", "2001:db8:1::2"},
		{"H3", "2001:db8:1::3"},
		{"H3", "2001:db8:2::1"},
		{"H3", "2001:db8:2::2"},
		{"H3", "2001:db8:3::2"},
		{"H3", "2001:db8:3::3"},

		// ping loopbacks

		{"R1", "2001:db8:0:192::1"},
		{"R1", "2001:db8:0:192::2"},
		{"R1", "2001:db8:0:192::3"},
		{"R1", "2001:db8:1:192::2"},

		{"R2", "2001:db8:0:192::1"},
		{"R2", "2001:db8:0:192::2"},
		{"R2", "2001:db8:0:192::3"},
		{"R2", "2001:db8:1:192::1"},

		{"H1", "2001:db8:0:192::2"},
		{"H1", "2001:db8:0:192::3"},
		{"H1", "2001:db8:1:192::1"},
		{"H1", "2001:db8:1:192::2"},

		{"H2", "2001:db8:0:192::1"},
		{"H2", "2001:db8:0:192::3"},
		{"H2", "2001:db8:1:192::1"},
		{"H2", "2001:db8:1:192::2"},

		{"H3", "2001:db8:1:192::1"},
		{"H3", "2001:db8:1:192::2"},
		{"H3", "2001:db8:0:192::2"},
		{"H3", "2001:db8:0:192::1"},
	} {
		assert.Nil(svi6.PingCmd(t, x.hostname, x.target))
		// assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
	}
}

type svi6OspfFlap struct{ *docker.Docket }

func (svi6OspfFlap) String() string { return "flap" }

func (svi6 svi6OspfFlap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range svi6.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := svi6.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = svi6.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
		}
	}
}

type svi6OspfAdminDown struct{ *docker.Docket }

func (svi6OspfAdminDown) String() string { return "admin-down" }

func (svi6 svi6OspfAdminDown) Test(t *testing.T) {
	assert := test.Assert{t}

	num_intf := 0
	for _, r := range svi6.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := svi6.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)
}
