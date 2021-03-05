// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"regexp"
	"testing"
	"time"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/docker"
)

func sliceVlanV6Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	sliceV6Test(t, "testdata/net6/slice/vlan/conf.yaml.tmpl")
}

func sliceV6Test(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		sliceV6Connectivity{docket},
		sliceV6Frr{docket},
		sliceV6Config{docket},
		sliceV6Neighbors{docket},
		sliceV6Routes{docket},
		sliceV6InterConnectivity{docket},
		sliceV6Isolation{docket},
		sliceV6Connectivity{docket},
		sliceV6Routes{docket},
		sliceV6InterConnectivity{docket},
		sliceV6Connectivity{docket},
		sliceV6Routes{docket},
		sliceV6InterConnectivity{docket})
}

type sliceV6Connectivity struct{ *docker.Docket }

func (sliceV6Connectivity) String() string { return "connectivity" }

func (slice sliceV6Connectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "2001:db8:0:1::2"},
		{"RA-1", "2001:db8:0:1::1"},
		{"RA-1", "2001:db8:0:2::3"},
		{"RA-2", "2001:db8:0:2::2"},
		{"RA-2", "2001:db8:0:3::4"},
		{"CA-2", "2001:db8:0:3::3"},
		{"CB-1", "2001:db8:0:1::2"},
		{"RB-1", "2001:db8:0:1::1"},
		{"RB-1", "2001:db8:0:2::3"},
		{"RB-2", "2001:db8:0:2::2"},
		{"RB-2", "2001:db8:0:3::4"},
		{"CB-2", "2001:db8:0:3::3"},
	} {
		assert.Nil(slice.PingCmd(t, x.hostname, x.target))
		//FIXME
		//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
		//	x.hostname)
	}
}

type sliceV6Frr struct{ *docker.Docket }

func (sliceV6Frr) String() string { return "frr" }

func (slice sliceV6Frr) Test(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)
	for _, r := range slice.Routers {
		assert.Comment("Checking FRR on", r.Hostname)
		out, err := slice.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.True(regexp.MustCompile(".*ospf6d.*").MatchString(out))
		assert.True(regexp.MustCompile(".*zebra.*").MatchString(out))
	}
}

type sliceV6Config struct{ *docker.Docket }

func (sliceV6Config) String() string { return "config" }

func (slice sliceV6Config) Test(t *testing.T) {
	assert := test.Assert{t}
	assert.Comment("configuring OSPF v3")

	for _, r := range slice.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := slice.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "ipv6 forwarding")
			assert.Nil(err)
			_, err = slice.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "interface "+intf, "-c", "ipv6 ospf6 network point-to-point")
			assert.Nil(err)
			_, err = slice.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "router ospf6", "-c", "interface "+intf+" area 0.0.0.0")
			assert.Nil(err)
			_, err = slice.ExecCmd(t, r.Hostname,
				"vtysh", "-c", "conf t", "-c", "router ospf6", "-c", "redistribute connected")
			assert.Nil(err)
		}
	}
}

type sliceV6Neighbors struct{ *docker.Docket }

func (sliceV6Neighbors) String() string { return "neighbors" }

func (slice sliceV6Neighbors) Test(t *testing.T) {
	assert := test.Assert{t}

	timeout := 120

	for _, x := range []struct {
		hostname string
		peer     string
	}{
		{"CA-1", "0.0.0.2"},
		{"RA-1", "0.0.0.1"},
		{"RA-1", "0.0.0.3"},
		{"RA-2", "0.0.0.2"},
		{"RA-2", "0.0.0.4"},
		{"CA-2", "0.0.0.3"},

		{"CB-1", "0.0.0.2"},
		{"RB-1", "0.0.0.1"},
		{"RB-1", "0.0.0.3"},
		{"RB-2", "0.0.0.2"},
		{"RB-2", "0.0.0.4"},
		{"CB-2", "0.0.0.3"},
	} {
		found := false
		for i := timeout; i > 0; i-- {
			out, err := slice.ExecCmd(t, x.hostname,
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
			t.Fatalf("No ospf neighbor found for %v peer %v", x.hostname, x.peer)
		}
	}
}

type sliceV6Routes struct{ *docker.Docket }

func (sliceV6Routes) String() string { return "routes" }

func (slice sliceV6Routes) Test(t *testing.T) {
	assert := test.Assert{t}

	test.Pause.Prompt("Stop")

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"CA-1", "2001:db8:0:3::/64"},
		{"CA-2", "2001:db8:0:1::/64"},
		{"CB-1", "2001:db8:0:3::/64"},
		{"CB-2", "2001:db8:0:1::/64"},
	} {
		found := false
		timeout := 120
		for i := timeout; i > 0; i-- {
			out, err := slice.ExecCmd(t, x.hostname,
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

type sliceV6InterConnectivity struct{ *docker.Docket }

func (sliceV6InterConnectivity) String() string { return "inter-connectivity" }

func (slice sliceV6InterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "2001:db8:0:3::4"}, // In slice A ping6 from CA-1 to CA-2
		{"CB-1", "2001:db8:0:3::4"}, // In slice B ping6 from CB-1 to CB-2
		{"CA-2", "2001:db8:0:1::1"}, // In slice A ping6 from CA-2 to CA-1
		{"CB-2", "2001:db8:0:1::1"}, // In slice B ping6 from CB-2 to CB-1

	} {
		assert.Nil(slice.PingCmd(t, x.hostname, x.target))
		//FIXME
		//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
		//	x.hostname)
	}
}

type sliceV6Isolation struct{ *docker.Docket }

func (sliceV6Isolation) String() string { return "isolation" }

func (slice sliceV6Isolation) Test(t *testing.T) {
	assert := test.Assert{t}

	// break slice B connectivity does not affect slice A
	r, err := docker.FindHost(slice.Config, "RB-2")
	assert.Nil(err)

	for _, i := range r.Intfs {
		intf := docker.IntfVlanName(i.Name, i.Vlan)
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "down", intf)
		assert.Nil(err)
	}
	// how do I do an anti match???
	// FIXME
	//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table", "RB-2")

	assert.Comment("Verify that slice B is broken")
	_, err = slice.ExecCmd(t, "CB-1", "ping6", "-c1", "2001:db8:0:3::4")
	assert.NonNil(err)

	assert.Comment("Verify that slice A is not affected")
	_, err = slice.ExecCmd(t, "CA-1", "ping6", "-c1", "2001:db8:0:3::4")
	assert.Nil(err)
	//FIXME
	//assert.Program(regexp.MustCompile("2001:db8:0:3::/64"),
	//	*Goes, "vnet", "show", "ip", "fib", "table", "RA-2")

	// bring RB-2 interfaces back up
	for _, i := range r.Intfs {
		intf := docker.IntfVlanName(i.Name, i.Vlan)
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "up", intf)
		assert.Nil(err)
	}

	// break slice A connectivity does not affect slice B
	r, err = docker.FindHost(slice.Config, "RA-2")
	assert.Nil(err)

	for _, i := range r.Intfs {
		intf := docker.IntfVlanName(i.Name, i.Vlan)
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "down", intf)
		assert.Nil(err)
	}
	// how do I do an anti match???
	// FIXME
	//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table", "RA-2")

	assert.Comment("Verify that slice A is broken")
	_, err = slice.ExecCmd(t, "CA-1", "ping6", "-c1", "2001:db8:0:3::4")
	assert.NonNil(err)

	ok := false
	assert.Comment("Verify that slice B is not affected")
	timeout := 120
	for i := timeout; i > 0; i-- {
		out, _ := slice.ExecCmd(t, "CB-1", "ping6", "-c1", "2001:db8:0:3::4")
		if !assert.MatchNonFatal(out, "1 received") {
			time.Sleep(1 * time.Second)
		} else {
			ok = true
			break
		}
	}
	if !ok {
		t.Error("Slice B ping6 failed")
	}
	//FIXME
	//assert.Program(regexp.MustCompile("2001:db8:0:3::/64"),
	//	*Goes, "vnet", "show", "ip", "fib", "table", "RB-2")

	// bring RA-1 interfaces back up
	for _, i := range r.Intfs {
		intf := docker.IntfVlanName(i.Name, i.Vlan)
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "up", intf)
		assert.Nil(err)
	}

}
