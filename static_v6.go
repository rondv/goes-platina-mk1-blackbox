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

func staticV6NetTest(t *testing.T) {
	staticV6Test(t, "testdata/net6/static/conf.yaml.tmpl")
}

func staticV6VlanTest(t *testing.T) {
	staticV6Test(t, "testdata/net6/static/vlan/conf.yaml.tmpl")
}

func staticV6Test(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		staticV6Connectivity{docket},
		staticV6Frr{docket},
		staticV6Routes{docket},
		staticV6InterConnectivity{docket},
		staticV6Flap{docket},
		staticV6InterConnectivity2{docket},
		staticV6PuntStress{docket},
		staticV6Blackhole{docket},
		staticV6AdminDown{docket})
}

type staticV6Connectivity struct{ *docker.Docket }

func (staticV6Connectivity) String() string { return "connectivity" }

func (staticV6 staticV6Connectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	test.Pause.Prompt("conditional pause")

	out, err := staticV6.ExecCmd(t, "RA-1", "vtysh", "-c",
		"conf t", "-c", "ipv6 route 2001:db8:0:0::1/128 2001:db8:0:1::1")
	assert.Comment("out = ", out, " err = ", err)
	staticV6.ExecCmd(t, "RA-2", "vtysh", "-c",
		"conf t", "-c", "ipv6 route 2001:db8:0:0::2/128 2001:db8:0:3::4")
	assert.Comment("out = ", out, " err = ", err)

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
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(staticV6.PingCmd(t, x.hostname, x.target))
	}
}

type staticV6Frr struct{ *docker.Docket }

func (staticV6Frr) String() string { return "frr" }

func (staticV6 staticV6Frr) Test(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)

	for _, r := range staticV6.Routers {
		assert.Comment("Checking FRR on", r.Hostname)
		out, err := staticV6.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.True(regexp.MustCompile(".*zebra.*").MatchString(out))
	}
}

type staticV6Routes struct{ *docker.Docket }

func (staticV6Routes) String() string { return "routes" }

func (staticV6 staticV6Routes) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range staticV6.Routers {

		assert.Comment("check for default route in container RIB",
			r.Hostname)
		out, err := staticV6.ExecCmd(t, r.Hostname, "vtysh", "-c",
			"show ipv6 route")
		assert.Nil(err)
		assert.Match(out, "S>\\* ::/0")

		assert.Comment("check for default route in container FIB",
			r.Hostname)
		out, err = staticV6.ExecCmd(t, r.Hostname, "ip", "-6", "route", "show")
		assert.Nil(err)
		assert.Match(out, "default")

		assert.Comment("check for default route in goes fib",
			r.Hostname)
		// FIXME
		//assert.Program(regexp.MustCompile("0.0.0.0/0"),
		//	*Goes, "vnet", "show", "ip", "fib", "table",
		//	r.Hostname)
	}
}

type staticV6InterConnectivity struct{ *docker.Docket }

func (staticV6InterConnectivity) String() string { return "inter-connectivity" }

func (staticV6 staticV6InterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	test.Pause.Prompt("conditional pause")

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "2001:db8:0:3::4"},
		// {"CA-1", "2001:db8:0:0::2"},
		{"CA-2", "2001:db8:0:1::1"},
		// {"CA-2", "2001:db8:0:0::1"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(staticV6.PingCmd(t, x.hostname, x.target))
		// FIXME
		//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
		//	x.hostname)
	}
}

type staticV6Flap struct{ *docker.Docket }

func (staticV6Flap) String() string { return "flap" }

func (staticV6 staticV6Flap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range staticV6.Routers {

		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := staticV6.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = staticV6.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			staticV6.ExecCmd(t, r.Hostname,
				"ip", "addr", "show", "dev", intf)
			assert.Program(*Goes, "fe1", "switch", "fib", "ip6")
		}
	}
}

type staticV6InterConnectivity2 struct{ *docker.Docket }

func (staticV6InterConnectivity2) String() string { return "inter-connectivity2" }

func (staticV6 staticV6InterConnectivity2) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "2001:db8:0:1::2"},
		{"RA-1", "2001:db8:0:1::1"},
		{"RA-1", "2001:db8:0:2::3"},
		// {"RA-1", "2001:db8:0:0::1"},
		{"RA-2", "2001:db8:0:2::2"},
		{"RA-2", "2001:db8:0:3::4"},
		// {"RA-2", "2001:db8:0:0::2"},
		{"CA-2", "2001:db8:0:3::3"},
		{"CA-1", "2001:db8:0:3::4"},
		// {"CA-1", "2001:db8:0:0::2"},
		{"CA-2", "2001:db8:0:1::1"},
		// {"CA-2", "2001:db8:0:0::1"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(staticV6.PingCmd(t, x.hostname, x.target))
		//FIXME
		//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
		//	x.hostname)
	}
}

type staticV6PuntStress struct{ *docker.Docket }

func (staticV6PuntStress) String() string { return "punt-stress" }

func (staticV6 staticV6PuntStress) Test(t *testing.T) {
	if testing.Short() || *test.DryRun {
		t.SkipNow()
	}

	assert := test.Assert{t}
	assert.Comment("Check punt stress with iperf3")

	done := make(chan bool, 1)

	go func(done chan bool) {
		staticV6.ExecCmd(t, "CA-2", "timeout", "15", "iperf3", "-s")
		done <- true
	}(done)

	time.Sleep(1 * time.Second)
	out, err := staticV6.ExecCmd(t, "CA-1", "iperf3", "-c", "2001:db8:0:3::4")

	r, err := regexp.Compile(`([0-9\.]+)\s+([GMK]?)bits/sec\s+receiver`)
	assert.Nil(err)
	result := r.FindStringSubmatch(out)
	if len(result) == 3 {
		assert.Commentf("iperf3 - %v %vbits/sec", result[1], result[2])
		assert.Comment("checking for not 0.00 bits/sec")
		assert.True(result[1] != "0.00")
	} else {
		assert.Fatalf("iperf3 regex failed to find rate [%v]", out)
	}
	<-done
}

type staticV6Blackhole struct{ *docker.Docket }

func (staticV6Blackhole) String() string { return "blackhole" }

func (staticV6 staticV6Blackhole) Test(t *testing.T) {
	if testing.Short() || *test.DryRun {
		t.SkipNow()
	}

	assert := test.Assert{t}
	assert.Comment("ping from CA-1 to CA-2 before blackhole")
	assert.Nil(staticV6.PingCmd(t, "CA-1", "2001:db8:0:3::4"))

	assert.Comment("Add blackhole /128 route")
	staticV6.ExecCmd(t, "RA-2", "ip", "-6", "route", "add", "blackhole",
		"2001:db8:0:3::4/128")
	time.Sleep(1 * time.Second)

	assert.Comment("ping should get swallowed by blackhole")
	assert.NonNil(staticV6.PingCmd(t, "CA-1", "2001:db8:0:3::4"))

	//FIXME
	//assert.Program(regexp.MustCompile("drop"),
	//	*Goes, "vnet", "show", "ip", "fib", "table",
	//	"RA-2")

	assert.Comment("Remove blackhole route")
	staticV6.ExecCmd(t, "RA-2", "ip", "-6", "route", "del", "blackhole",
		"2001:db8:0:3::4/128")
	time.Sleep(1 * time.Second)

	assert.Comment("ping should work again")
	assert.Nil(staticV6.PingCmd(t, "CA-1", "2001:db8:0:3::4"))

	assert.Comment("Add blackhole route to dummy")
	staticV6.ExecCmd(t, "RA-2", "ip", "route", "add",
		"2001:db8:0:0::/64", "via", "2001:db8:0:3::4")
	assert.Comment("ping dummy on CA-2")
	//assert.Nil(staticV6.PingCmd(t, "CA-1", "2001:db8:0:0::2"))
	//FIXME
	//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
	//	"RA-2")

	assert.Comment("Add blackhole for dummy")
	staticV6.ExecCmd(t, "RA-1", "ip", "-6", "route", "add",
		"blackhole", "2001:db8:0:0::/64")
	time.Sleep(1 * time.Second)
	//FIXME
	//assert.Program(regexp.MustCompile("drop"),
	//	*Goes, "vnet", "show", "ip", "fib", "table",
	//	"RA-1")

	assert.Comment("Now ping should fail")
	//assert.NonNil(staticV6.PingCmd(t, "CA-1", "2001:db8:0:0::2"))

	assert.Comment("Remove blackhole for dummy")
	staticV6.ExecCmd(t, "RA-1", "ip", "-6", "route", "del",
		"blackhole", "2001:db8:0:0::/64")
	time.Sleep(1 * time.Second)
	//FIXME
	//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
	//	"RA-1")

	assert.Comment("Now ping should work again")
	//assert.Nil(staticV6.PingCmd(t, "CA-1", "2001:db8:0:0::2"))
}

type staticV6AdminDown struct{ *docker.Docket }

func (staticV6AdminDown) String() string { return "admin down" }

func (staticV6 staticV6AdminDown) Test(t *testing.T) {
	if testing.Short() || *test.DryRun {
		t.SkipNow()
	}

	assert := test.Assert{t}
	num_intf := 0
	for _, r := range staticV6.Routers {
		for _, i := range r.Intfs {
			intf := docker.IntfVlanName(i.Name, i.Vlan)
			_, err := staticV6.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			num_intf++
		}
	}
	AssertNoAdjacencies(t)

}
