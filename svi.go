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

func sviNetTest(t *testing.T) {
	t.Run("ospf", sviNetOspfTest)
	t.Run("isis", sviNetIsisTest)
}

func sviNetOspfTest(t *testing.T) {
	sviOspfTest(t, "testdata/svi/ospf/conf.yaml.tmpl")
}

func sviNetIsisTest(t *testing.T) {
	sviIsisTest(t, "testdata/svi/isis/conf.yaml.tmpl")
}

func sviOspfTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		sviCarrier{docket},
		sviConnectivity{docket},
		sviOspfDaemons{docket},
		sviOspfConfig{docket},
		sviOspfNeighbors{docket},
		sviOspfRoutes{docket},
		sviOspfInterConnectivity{docket},
		sviOspfFlap{docket},
		sviConnectivity{docket},
		sviOspfNeighbors{docket},
		sviOspfRoutes{docket},
		sviOspfInterConnectivity{docket},
		sviOspfAdminDown{docket},
	)
}

type sviConnectivity struct{ *docker.Docket }

func (sviConnectivity) String() string { return "connectivity" }

func (svi sviConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"H1", "1.0.0.1"},
		{"H1", "1.0.0.3"},
		{"H2", "1.0.0.1"},
		{"H2", "1.0.0.2"},
		{"R1", "1.0.0.2"},
		{"R1", "1.0.0.3"},
		{"R1", "2.0.0.2"},
		{"R2", "2.0.0.1"},
		{"R2", "3.0.0.3"},
		{"H3", "3.0.0.2"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(svi.PingCmd(t, x.hostname, x.target))
	}
}

type sviCarrier struct{ *docker.Docket }

func (sviCarrier) String() string { return "carrier" }

func (svi sviCarrier) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range svi.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			assert.Comment("check carrier for", r.Hostname,
				"on", intf)
			assert.Nil(test.Carrier(r.Hostname, intf))
		}
	}
}

type sviOspfDaemons struct{ *docker.Docket }

func (sviOspfDaemons) String() string { return "daemons" }

func (svi sviOspfDaemons) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range svi.Routers {
		assert.Comment("ing FRR on", r.Hostname)
		out, err := svi.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.Match(out, ".*ospfd.*")
		assert.Match(out, ".*zebra.*")
	}
}

type sviOspfConfig struct{ *docker.Docket }

func (sviOspfConfig) String() string { return "config" }

func (svi sviOspfConfig) Test(t *testing.T) {
	assert := test.Assert{t}
	assert.Comment("configuring OSPF v2")

	for _, r := range svi.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := svi.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "ip forwarding")
			assert.Nil(err)
			if r.Hostname != "H1" && r.Hostname != "H2" && intf != "br0" && intf != "dummy0" {
				_, err = svi.ExecCmd(t, r.Hostname,
					"vtysh", "-c", "conf t", "-c", "interface "+intf, "-c", "ip ospf network point-to-point")
				assert.Nil(err)
			}
			_, err = svi.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "interface "+intf, "-c", "ip ospf area 0.0.0.0")
			assert.Nil(err)
			_, err = svi.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "router ospf", "-c", "redistribute connected")
			assert.Nil(err)
		}
	}
}

type sviOspfNeighbors struct{ *docker.Docket }

func (sviOspfNeighbors) String() string { return "neighbors" }

func (svi sviOspfNeighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	timeout := 120

	for _, x := range []struct {
		hostname string
		peer     string
	}{
		{"H1", "0.0.0.1"},
		{"H1", "1.0.0.2"},
		{"H2", "0.0.0.1"},
		{"H2", "1.0.0.1"},
		{"R1", "1.0.0.2"},
		{"R1", "1.0.0.3"},
		{"R1", "2.0.0.2"},
		{"R2", "2.0.0.1"},
		{"R2", "3.0.0.3"},
		{"H3", "0.0.0.2"},
	} {
		found := false
		for i := timeout; i > 0; i-- {
			out, err := svi.ExecCmd(t, x.hostname,
				"vtysh", "-c", "show ip ospf neighbor")
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

type sviOspfRoutes struct{ *docker.Docket }

func (sviOspfRoutes) String() string { return "routes" }

func (svi sviOspfRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"H1", "2.0.0.0/24"},
		{"H1", "3.0.0.0/24"},
		{"H2", "2.0.0.0/24"},
		{"H2", "3.0.0.0/24"},
		{"R1", "2.0.0.0/24"},
		{"R1", "3.0.0.0/24"},
		{"R2", "1.0.0.0/24"},
		{"R2", "2.0.0.0/24"},
		{"H3", "1.0.0.0/24"},
		{"H3", "2.0.0.0/24"},

		// loopbacks
		{"H1", "192.168.0.2"},
		{"H1", "192.168.0.3"},
		{"H1", "192.168.1.1"},
		{"H1", "192.168.1.2"},

		{"H2", "192.168.0.1"},
		{"H2", "192.168.0.3"},
		{"H2", "192.168.1.1"},
		{"H2", "192.168.1.2"},

		{"R1", "192.168.0.1"},
		{"R1", "192.168.0.2"},
		{"R1", "192.168.0.3"},
		{"R1", "192.168.1.2"},

		{"R2", "192.168.0.1"},
		{"R2", "192.168.0.2"},
		{"R2", "192.168.0.3"},
		{"R2", "192.168.1.1"},

		{"H3", "192.168.0.1"},
		{"H3", "192.168.0.2"},
		{"H3", "192.168.1.1"},
		{"H3", "192.168.1.2"},
	} {
		found := false
		timeout := 60
		for i := timeout; i > 0; i-- {
			out, err := svi.ExecCmd(t, x.hostname,
				"ip", "route", "show", x.route)
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

type sviOspfInterConnectivity struct{ *docker.Docket }

func (sviOspfInterConnectivity) String() string { return "inter-connectivity" }

func (svi sviOspfInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"H1", "3.0.0.3"},
		{"H2", "3.0.0.3"},
		{"R1", "1.0.0.2"},
		{"R1", "1.0.0.3"},
		{"R1", "2.0.0.2"},
		{"R1", "3.0.0.2"},
		{"R1", "3.0.0.3"},
		{"R2", "1.0.0.1"},
		{"R2", "1.0.0.2"},
		{"R2", "1.0.0.3"},
		{"R2", "2.0.0.1"},
		{"R2", "3.0.0.3"},
		{"H1", "192.168.0.2"},
		{"H1", "192.168.0.3"},
		{"H1", "192.168.0.1"},
		{"H1", "192.168.0.3"},
		{"H3", "192.168.1.1"},
		{"H3", "192.168.0.2"},
		{"H3", "192.168.0.1"},
	} {
		assert.Nil(svi.PingCmd(t, x.hostname, x.target))
		assert.Program(*Goes, "fe1", "switch", "fib")
	}
}

type sviOspfFlap struct{ *docker.Docket }

func (sviOspfFlap) String() string { return "flap" }

func (svi sviOspfFlap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range svi.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := svi.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = svi.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(*Goes, "fe1", "switch", "fib")
		}
	}
}

type sviOspfAdminDown struct{ *docker.Docket }

func (sviOspfAdminDown) String() string { return "admin-down" }

func (svi sviOspfAdminDown) Test(t *testing.T) {
	assert := test.Assert{t}

	num_intf := 0
	for _, r := range svi.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := svi.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)
}

func sviIsisTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		sviIsisConnectivity{docket},
		sviIsisDaemons{docket},
		sviIsisAddIntfConf{docket},
		sviIsisNeighbors{docket},
		sviIsisRoutes{docket},
		sviIsisInterConnectivity{docket},
		sviIsisFlap{docket},
		sviIsisConnectivity{docket},
		sviIsisAdminDown{docket})
}

type sviIsisConnectivity struct{ *docker.Docket }

func (sviIsisConnectivity) String() string { return "connectivity" }

func (frr sviIsisConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		host   string
		target string
	}{
		{"H1", "1.0.0.1"},
		{"H1", "1.0.0.3"},
		{"H2", "1.0.0.1"},
		{"H2", "1.0.0.2"},
		{"R1", "1.0.0.2"},
		{"R1", "1.0.0.3"},
		{"R1", "2.0.0.2"},
		{"R2", "2.0.0.1"},
		{"R2", "3.0.0.3"},
		{"H3", "3.0.0.2"},
	} {
		assert.Nil(frr.PingCmd(t, x.host, x.target))
	}
}

type sviIsisDaemons struct{ *docker.Docket }

func (sviIsisDaemons) String() string { return "daemons" }

func (frr sviIsisDaemons) Test(t *testing.T) {
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

type sviIsisAddIntfConf struct{ *docker.Docket }

func (sviIsisAddIntfConf) String() string { return "add-intf-conf" }

func (frr sviIsisAddIntfConf) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := frr.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t",
				"-c", "interface "+intf,
				"-c", "ip router isis "+r.Hostname)
			assert.Nil(err)
		}
	}
}

type sviIsisNeighbors struct{ *docker.Docket }

func (sviIsisNeighbors) String() string { return "neighbors" }

func (svi sviIsisNeighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		peer     string
		address  string
	}{
		{"H1", "R1", "1.0.0.1"},
		{"H1", "H2", "1.0.0.3"},
		{"H2", "R1", "1.0.0.1"},
		{"H2", "H1", "1.0.0.2"},
		{"R1", "H1", "1.0.0.2"},
		{"R1", "H2", "1.0.0.3"},
		{"R1", "R2", "2.0.0.2"},
		{"R2", "R1", "2.0.0.1"},
		{"R2", "H3", "3.0.0.3"},
		{"H3", "R2", "3.0.0.2"},
	} {
		timeout := 60
		found := false
		for i := timeout; i > 0; i-- {
			out, err := svi.ExecCmd(t, x.hostname,
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

type sviIsisRoutes struct{ *docker.Docket }

func (sviIsisRoutes) String() string { return "routes" }

func (frr sviIsisRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"H1", "2.0.0.0/24"},
		{"H1", "3.0.0.0/24"},
		{"H2", "2.0.0.0/24"},
		{"H2", "3.0.0.0/24"},
		{"R1", "2.0.0.0/24"},
		{"R1", "3.0.0.0/24"},
		{"R2", "1.0.0.0/24"},
		{"R2", "2.0.0.0/24"},
		{"H3", "1.0.0.0/24"},
		{"H3", "2.0.0.0/24"},

		// loopbacks
		{"H1", "192.168.0.2"},
		{"H1", "192.168.0.3"},
		{"H1", "192.168.1.1"},
		{"H1", "192.168.1.2"},

		{"H2", "192.168.0.1"},
		{"H2", "192.168.0.3"},
		{"H2", "192.168.1.1"},
		{"H2", "192.168.1.2"},

		{"R1", "192.168.0.1"},
		{"R1", "192.168.0.2"},
		{"R1", "192.168.0.3"},
		{"R1", "192.168.1.2"},

		{"R2", "192.168.0.1"},
		{"R2", "192.168.0.2"},
		{"R2", "192.168.0.3"},
		{"R2", "192.168.1.1"},

		{"H3", "192.168.0.1"},
		{"H3", "192.168.0.2"},
		{"H3", "192.168.1.1"},
		{"H3", "192.168.1.2"},
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

type sviIsisInterConnectivity struct{ *docker.Docket }

func (sviIsisInterConnectivity) String() string { return "inter-connectivity" }

func (frr sviIsisInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"H1", "3.0.0.3"},
		{"H2", "3.0.0.3"},
		{"R1", "1.0.0.2"},
		{"R1", "1.0.0.3"},
		{"R1", "2.0.0.2"},
		{"R1", "3.0.0.2"},
		{"R1", "3.0.0.3"},
		{"R2", "1.0.0.1"},
		{"R2", "1.0.0.2"},
		{"R2", "1.0.0.3"},
		{"R2", "2.0.0.1"},
		{"R2", "3.0.0.3"},
		{"H1", "192.168.0.2"},
		{"H1", "192.168.0.3"},
		{"H1", "192.168.0.1"},
		{"H1", "192.168.0.3"},
		{"H3", "192.168.1.1"},
		{"H3", "192.168.0.2"},
		{"H3", "192.168.0.1"},
	} {
		assert.Nil(frr.PingCmd(t, x.hostname, x.target))
		assert.Program(*Goes, "fe1", "switch", "fib")
	}
}

type sviIsisFlap struct{ *docker.Docket }

func (sviIsisFlap) String() string { return "flap" }

func (frr sviIsisFlap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(*Goes, "fe1", "switch", "fib")
		}
	}
}

type sviIsisAdminDown struct{ *docker.Docket }

func (sviIsisAdminDown) String() string { return "admin-down" }

func (frr sviIsisAdminDown) Test(t *testing.T) {
	assert := test.Assert{t}

	num_intf := 0
	for _, r := range frr.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := frr.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)
}
