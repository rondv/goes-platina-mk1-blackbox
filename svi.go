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
	sviOspfTest(t, "testdata/svi/ospf/conf.yaml.tmpl")
}

func sviOspfTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		sviCarrier{docket},
		sviConnectivity{docket},
		sviOspfDaemons{docket},
		sviOspfNeighbors{docket},
		sviOspfRoutes{docket},
		sviOspfInterConnectivity{docket},
		sviOspfFlap{docket},
		sviConnectivity{docket},
		sviOspfAdminDown{docket},
	)
}

type sviConnectivity struct{ *docker.Docket }

func (sviConnectivity) String() string { return "connectivity" }

func (svi sviConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	test.Pause.Prompt("stop")

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"H1", "1.0.0.1"},
		{"H2", "1.0.0.1"},
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

type sviOspfNeighbors struct{ *docker.Docket }

func (sviOspfNeighbors) String() string { return "neighbors" }

func (svi sviOspfNeighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	timeout := 120

	for _, x := range []struct {
		hostname string
		peer     string
	}{
		{"R2", "2.0.0.1"},
		{"R2", "3.0.0.3"},
		{"R1", "2.0.0.2"},
		{"R1", "1.0.0.2"},
		{"R1", "1.0.0.3"},
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
		{"R2", "1.0.0.0/24"},
		{"R2", "2.0.0.0/24"},
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
		{"H3", "192.168.1.1"},
		{"H3", "192.168.0.2"},
		//{"H3", "192.168.0.1"},
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
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
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
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := svi.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)
}
