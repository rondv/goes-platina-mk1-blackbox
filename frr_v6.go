// Copyright © 2002 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"testing"
	"time"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/docker"
)

func frrNetV6Test(t *testing.T) {
	t.Run("ospf", frrNetV6OspfTest)
	test.SkipIfDryRun(t)
}

func frrVlanV6Test(t *testing.T) {
	t.Run("ospf", frrVlanV6OspfTest)
	test.SkipIfDryRun(t)
}

func frrNetV6BgpTest(t *testing.T) {
	frrV6BgpTest(t, "testdata/frr6/bgp/conf.yaml.tmpl")
}

func frrVlanV6BgpTest(t *testing.T) {
	frrV6BgpTest(t, "testdata/frr6/bgp/vlan/conf.yaml.tmpl")
}

func frrV6BgpTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		frrV6BgpConnectivity{docket},
		frrV6BgpDaemons{docket},
		frrV6BgpBfd{docket},
		frrV6BgpNeighbors{docket},
		frrV6BgpRoutes{docket},
		frrV6BgpInterConnectivity{docket},
		frrV6BgpFlap{docket},
		frrV6BgpConnectivity{docket},
		frrV6BgpAdminDown{docket})
}

func frrNetV6OspfTest(t *testing.T) {
	frrV6OspfTest(t, "testdata/frr6/ospf/conf.yaml.tmpl")
}

func frrVlanV6OspfTest(t *testing.T) {
	frrV6OspfTest(t, "testdata/frr6/ospf/vlan/conf.yaml.tmpl")
}

func frrV6OspfTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		frrV6OspfCarrier{docket},
		frrV6OspfConnectivity{docket},
		frrV6OspfDaemons{docket},
		frrV6OspfConfig{docket},
		frrV6OspfNeighbors{docket},
		frrV6OspfRoutes{docket},
		frrV6OspfInterConnectivity{docket},
		frrV6OspfFlap{docket},
		frrV6OspfConnectivity{docket},
		frrV6OspfAdminDown{docket})
}

func frrNetV6IsisTest(t *testing.T) {
	frrV6IsisTest(t, "testdata/frr6/isis/conf.yaml.tmpl")
}

func frrVlanV6IsisTest(t *testing.T) {
	frrV6IsisTest(t, "testdata/frr6/isis/vlan/conf.yaml.tmpl")
}

func frrV6IsisTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		frrV6IsisConnectivity{docket},
		frrV6IsisDaemons{docket},
		frrV6IsisAddIntfConf{docket},
		frrV6IsisNeighbors{docket},
		frrV6IsisRoutes{docket},
		frrV6IsisInterConnectivity{docket},
		frrV6IsisFlap{docket},
		frrV6IsisConnectivity{docket},
		frrV6IsisAdminDown{docket})
}

type frrV6BgpConnectivity struct{ *docker.Docket }

func (frrV6BgpConnectivity) String() string { return "connectivity" }

func (frr frrV6BgpConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	test.Pause.Prompt("Stop")
	for _, x := range []struct {
		host   string
		target string
	}{
		{"R1", "2001:db8:0:120::10"},
		{"R1", "2001:db8:0:150::4"},
		{"R2", "2001:db8:0:222::2"},
		{"R2", "2001:db8:0:120::5"},
		{"R3", "2001:db8:0:222::10"},
		{"R3", "2001:db8:0:111::4"},
		{"R4", "2001:db8:0:111::2"},
		{"R4", "2001:db8:0:150::5"},
	} {
		assert.Nil(frr.PingCmd(t, x.host, x.target))
	}
}

type frrV6BgpDaemons struct{ *docker.Docket }

func (frrV6BgpDaemons) String() string { return "daemons" }

func (frr frrV6BgpDaemons) Test(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)
	for _, r := range frr.Routers {
		assert.Comment("ing FRR on", r.Hostname)
		out, err := frr.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.Match(out, ".*bgpd.*")
		assert.Match(out, ".*zebra.*")
	}
}

type frrV6BgpBfd struct{ *docker.Docket }

func (frrV6BgpBfd) String() string { return "bfd" }

func (frr frrV6BgpBfd) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		peer     string
	}{
		{"R1", "2001:db8:0:120::10"},
		{"R1", "2001:db8:0:150::4"},
		{"R2", "2001:db8:0:120::5"},
		{"R2", "2001:db8:0:222::2"},
		{"R3", "2001:db8:0:222::10"},
		{"R3", "2001:db8:0:111::4"},
		{"R4", "2001:db8:0:111::2"},
		{"R4", "2001:db8:0:150::5"},
	} {
		out, err := frr.ExecCmd(t, x.hostname,
			"vtysh", "-c", "show bfd peer "+x.peer)
		assert.Nil(err)
		assert.Match(out, ".*Status: up.*")
	}
}

type frrV6BgpNeighbors struct{ *docker.Docket }

func (frrV6BgpNeighbors) String() string { return "neighbors" }

func (frr frrV6BgpNeighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		peer     string
	}{
		{"R1", "2001:db8:0:120::10"},
		{"R1", "2001:db8:0:150::4"},
		{"R2", "2001:db8:0:120::5"},
		{"R2", "2001:db8:0:222::2"},
		{"R3", "2001:db8:0:222::10"},
		{"R3", "2001:db8:0:111::4"},
		{"R4", "2001:db8:0:111::2"},
		{"R4", "2001:db8:0:150::5"},
	} {
		found := false
		timeout := 120

		for i := timeout; i > 0; i-- {
			out, err := frr.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show ip bgp neighbor "+x.peer)
			assert.Nil(err)
			if !assert.MatchNonFatal(out, ".*state = Established.*") {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No bgp peer established for %v", x.hostname)
		}
	}
}

type frrV6BgpRoutes struct{ *docker.Docket }

func (frrV6BgpRoutes) String() string { return "routes" }

func (frr frrV6BgpRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"R1", "2001:db8:0:222::/64"},
		{"R1", "2001:db8:0:111::/64"},
		{"R2", "2001:db8:0:150::/64"},
		{"R2", "2001:db8:0:111::/64"},
		{"R3", "2001:db8:0:120::/64"},
		{"R3", "2001:db8:0:150::/64"},
		{"R4", "2001:db8:0:120::/64"},
		{"R4", "2001:db8:0:222::/64"},
	} {
		found := false
		timeout := 60
		for i := timeout; i > 0; i-- {
			out, err := frr.ExecCmd(t, x.hostname,
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
			t.Fatalf("No bgp route for %v: %v", x.hostname, x.route)
		}
	}
}

type frrV6BgpInterConnectivity struct{ *docker.Docket }

func (frrV6BgpInterConnectivity) String() string { return "inter-connectivity" }

func (frr frrV6BgpInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"R1", "2001:db8:0:222::2"},
		{"R1", "2001:db8:0:111::2"},
		{"R2", "2001:db8:0:111::4"},
		{"R2", "2001:db8:0:150::4"},
		{"R3", "2001:db8:0:120::5"},
		{"R3", "2001:db8:0:150::5"},
		{"R4", "2001:db8:0:120::10"},
		{"R4", "2001:db8:0:222::10"},
	} {
		err := frr.PingCmd(t, x.hostname, x.target)
		assert.Nil(err)
		assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
	}
}

type frrV6BgpFlap struct{ *docker.Docket }

func (frrV6BgpFlap) String() string { return "flap" }

func (frr frrV6BgpFlap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
		}
	}
}

type frrV6BgpAdminDown struct{ *docker.Docket }

func (frrV6BgpAdminDown) String() string { return "admin-down" }

func (frr frrV6BgpAdminDown) Test(t *testing.T) {
	assert := test.Assert{t}

	num_intf := 0
	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)
}

type frrV6OspfCarrier struct{ *docker.Docket }

func (frrV6OspfCarrier) String() string { return "carrier" }

func (frr frrV6OspfCarrier) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range frr.Routers {
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

type frrV6OspfConnectivity struct{ *docker.Docket }

func (frrV6OspfConnectivity) String() string { return "connectivity" }

func (frr frrV6OspfConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		host   string
		target string
	}{
		{"R1", "2001:db8:0:120::10"},
		{"R1", "2001:db8:0:150::4"},
		{"R2", "2001:db8:0:222::2"},
		{"R2", "2001:db8:0:120::5"},
		{"R3", "2001:db8:0:222::10"},
		{"R3", "2001:db8:0:111::4"},
		{"R4", "2001:db8:0:111::2"},
		{"R4", "2001:db8:0:150::5"},
	} {
		assert.Nil(frr.PingCmd(t, x.host, x.target))
	}
}

type frrV6OspfDaemons struct{ *docker.Docket }

func (frrV6OspfDaemons) String() string { return "daemons" }

func (frr frrV6OspfDaemons) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range frr.Routers {
		assert.Comment("ing FRR on", r.Hostname)
		out, err := frr.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.Match(out, ".*ospf6d.*")
		assert.Match(out, ".*zebra.*")
	}
}

type frrV6OspfConfig struct{ *docker.Docket }

func (frrV6OspfConfig) String() string { return "neighbors" }

func (frr frrV6OspfConfig) Test(t *testing.T) {
	assert := test.Assert{t}
	assert.Comment("configuring OSPF v3")

	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "ipv6 forwarding")
			assert.Nil(err)
			_, err = frr.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "router ospf6", "-c", "interface "+intf+" area 0.0.0.0")
			assert.Nil(err)
			_, err = frr.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "router ospf6", "-c", "redistribute connected")
			assert.Nil(err)
		}
	}
}

type frrV6OspfNeighbors struct{ *docker.Docket }

func (frrV6OspfNeighbors) String() string { return "neighbors" }

func (frr frrV6OspfNeighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	timeout := 120

	for _, x := range []struct {
		hostname string
		peer     string
	}{
		{"R1", "0.0.0.2"},
		{"R1", "0.0.0.4"},
		{"R2", "0.0.0.1"},
		{"R2", "0.0.0.3"},
		{"R3", "0.0.0.2"},
		{"R3", "0.0.0.4"},
		{"R4", "0.0.0.3"},
		{"R4", "0.0.0.1"},
	} {
		found := false
		for i := timeout; i > 0; i-- {
			out, err := frr.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show ipv6 ospf6 neighbor")
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

type frrV6OspfRoutes struct{ *docker.Docket }

func (frrV6OspfRoutes) String() string { return "routes" }

func (frr frrV6OspfRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	test.Pause.Prompt("Check IPv6 OSPF routes")

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"R1", "2001:db8:0:222::/64"},
		{"R1", "2001:db8:0:111::/64"},
		{"R2", "2001:db8:0:150::/64"},
		{"R2", "2001:db8:0:111::/64"},
		{"R3", "2001:db8:0:120::/64"},
		{"R3", "2001:db8:0:150::/64"},
		{"R4", "2001:db8:0:120::/64"},
		{"R4", "2001:db8:0:222::/64"},
	} {
		found := false
		timeout := 60
		for i := timeout; i > 0; i-- {
			out, err := frr.ExecCmd(t, x.hostname,
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

type frrV6OspfInterConnectivity struct{ *docker.Docket }

func (frrV6OspfInterConnectivity) String() string { return "inter-connectivity" }

func (frr frrV6OspfInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"R1", "2001:db8:0:222::2"},
		{"R1", "2001:db8:0:111::2"},
		{"R2", "2001:db8:0:111::4"},
		{"R2", "2001:db8:0:150::4"},
		{"R3", "2001:db8:0:120::5"},
		{"R3", "2001:db8:0:150::5"},
		{"R4", "2001:db8:0:120::10"},
		{"R4", "2001:db8:0:222::10"},
	} {
		assert.Nil(frr.PingCmd(t, x.hostname, x.target))
		assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
	}
}

type frrV6OspfFlap struct{ *docker.Docket }

func (frrV6OspfFlap) String() string { return "flap" }

func (frr frrV6OspfFlap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
		}
	}
}

type frrV6OspfAdminDown struct{ *docker.Docket }

func (frrV6OspfAdminDown) String() string { return "admin-down" }

func (frr frrV6OspfAdminDown) Test(t *testing.T) {
	assert := test.Assert{t}

	num_intf := 0
	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)
}

type frrV6IsisConnectivity struct{ *docker.Docket }

func (frrV6IsisConnectivity) String() string { return "connectivity" }

func (frr frrV6IsisConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		host   string
		target string
	}{
		{"R1", "2001:db8:0:120::10"},
		{"R1", "2001:db8:0:150::4"},
		{"R2", "2001:db8:0:222::2"},
		{"R2", "2001:db8:0:120::5"},
		{"R3", "2001:db8:0:222::10"},
		{"R3", "2001:db8:0:111::4"},
		{"R4", "2001:db8:0:111::2"},
		{"R4", "2001:db8:0:150::5"},
	} {
		assert.Nil(frr.PingCmd(t, x.host, x.target))
	}
}

type frrV6IsisDaemons struct{ *docker.Docket }

func (frrV6IsisDaemons) String() string { return "daemons" }

func (frr frrV6IsisDaemons) Test(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)

	for _, r := range frr.Routers {
		assert.Comment("ing FRR on", r.Hostname)
		out, err := frr.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.Match(out, ".*isisd.*")
		assert.Match(out, ".*zebra.*")
	}
}

type frrV6IsisAddIntfConf struct{ *docker.Docket }

func (frrV6IsisAddIntfConf) String() string { return "add-intf-conf" }

func (frr frrV6IsisAddIntfConf) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t",
				"-c", "interface "+intf,
				"-c", "ip router isis "+r.Hostname)
			assert.Nil(err)
		}
	}
}

type frrV6IsisNeighbors struct{ *docker.Docket }

func (frrV6IsisNeighbors) String() string { return "neighbors" }

func (frr frrV6IsisNeighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		peer     string
		address  string
	}{
		{"R1", "R2", "2001:db8:0:120::10"},
		{"R1", "R4", "2001:db8:0:150::4"},
		{"R2", "R1", "2001:db8:0:120::5"},
		{"R2", "R3", "2001:db8:0:222::2"},
		{"R3", "R2", "2001:db8:0:222::10"},
		{"R3", "R4", "2001:db8:0:111::4"},
		{"R4", "R3", "2001:db8:0:111::2"},
		{"R4", "R1", "2001:db8:0:150::5"},
	} {
		timeout := 60
		found := false
		for i := timeout; i > 0; i-- {
			out, err := frr.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show isis neighbor "+x.peer)
			assert.Nil(err)
			if !assert.MatchNonFatal(out, x.address) {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No isis neighbor for %v: %v",
				x.hostname, x.peer)
		}
	}
}

type frrV6IsisRoutes struct{ *docker.Docket }

func (frrV6IsisRoutes) String() string { return "routes" }

func (frr frrV6IsisRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"R1", "2001:db8:0:222::/64"},
		{"R1", "2001:db8:0:111::/64"},
		{"R2", "2001:db8:0:150::/64"},
		{"R2", "2001:db8:0:111::/64"},
		{"R3", "2001:db8:0:120::/64"},
		{"R3", "2001:db8:0:150::/64"},
		{"R4", "2001:db8:0:120::/64"},
		{"R4", "2001:db8:0:222::/64"},
	} {
		found := false
		timeout := 60
		for i := timeout; i > 0; i-- {
			out, err := frr.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show ip route isis")
			assert.Nil(err)
			if !assert.MatchNonFatal(out, x.route) {
				time.Sleep(1 * time.Second)
			} else {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("No isis route for %v: %v", x.hostname, x.route)
		}
	}
}

type frrV6IsisInterConnectivity struct{ *docker.Docket }

func (frrV6IsisInterConnectivity) String() string { return "inter-connectivity" }

func (frr frrV6IsisInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"R1", "2001:db8:0:222::2"},
		{"R1", "2001:db8:0:111::2"},
		{"R2", "2001:db8:0:111::4"},
		{"R2", "2001:db8:0:150::4"},
		{"R3", "2001:db8:0:120::5"},
		{"R3", "2001:db8:0:150::5"},
		{"R4", "2001:db8:0:120::10"},
		{"R4", "2001:db8:0:222::10"},
	} {
		assert.Nil(frr.PingCmd(t, x.hostname, x.target))
		assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
	}
}

type frrV6IsisFlap struct{ *docker.Docket }

func (frrV6IsisFlap) String() string { return "flap" }

func (frr frrV6IsisFlap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
		}
	}
}

type frrV6IsisAdminDown struct{ *docker.Docket }

func (frrV6IsisAdminDown) String() string { return "admin-down" }

func (frr frrV6IsisAdminDown) Test(t *testing.T) {
	assert := test.Assert{t}

	num_intf := 0
	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)
}
