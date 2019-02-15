// Copyright © 2015-2018 Platina Systems, Inc. All rights reserved.
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

func staticNetTest(t *testing.T) {
	staticTest(t, "testdata/net/static/conf.yaml.tmpl")
}

func staticVlanTest(t *testing.T) {
	staticTest(t, "testdata/net/static/vlan/conf.yaml.tmpl")
}

func staticTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		staticConnectivity{docket},
		staticFrr{docket},
		staticRoutes{docket},
		staticInterConnectivity{docket},
		staticFlap{docket},
		staticInterConnectivity2{docket},
		staticPuntStress{docket})
}

type staticConnectivity struct{ *docker.Docket }

func (staticConnectivity) String() string { return "connectivity" }

func (static staticConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "10.1.0.2"},
		{"RA-1", "10.1.0.1"},
		{"RA-1", "10.2.0.3"},
		{"RA-1", "192.168.0.1"},
		{"RA-2", "10.2.0.2"},
		{"RA-2", "10.3.0.4"},
		{"RA-2", "192.168.0.2"},
		{"CA-2", "10.3.0.3"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(static.PingCmd(t, x.hostname, x.target))
	}
}

type staticFrr struct{ *docker.Docket }

func (staticFrr) String() string { return "frr" }

func (static staticFrr) Test(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)

	for _, r := range static.Routers {
		assert.Comment("Checking FRR on", r.Hostname)
		out, err := static.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.True(regexp.MustCompile(".*zebra.*").MatchString(out))
	}
}

type staticRoutes struct{ *docker.Docket }

func (staticRoutes) String() string { return "routes" }

func (static staticRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, r := range static.Routers {

		assert.Comment("check for default route in container RIB",
			r.Hostname)
		out, err := static.ExecCmd(t, r.Hostname, "vtysh", "-c",
			"show ip route")
		assert.Nil(err)
		assert.Match(out, "S>\\* 0.0.0.0/0")

		assert.Comment("check for default route in container FIB",
			r.Hostname)
		out, err = static.ExecCmd(t, r.Hostname, "ip", "route", "show")
		assert.Nil(err)
		assert.Match(out, "default")

		assert.Comment("check for default route in goes fib",
			r.Hostname)
		assert.Program(regexp.MustCompile("0.0.0.0/0"),
			*Goes, "vnet", "show", "ip", "fib", "table",
			r.Hostname)
	}
}

type staticInterConnectivity struct{ *docker.Docket }

func (staticInterConnectivity) String() string { return "inter-connectivity" }

func (static staticInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "10.3.0.4"},
		{"CA-1", "192.168.0.2"},
		{"CA-2", "10.1.0.1"},
		{"CA-2", "192.168.0.1"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(static.PingCmd(t, x.hostname, x.target))
		assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
			x.hostname)
	}
}

type staticFlap struct{ *docker.Docket }

func (staticFlap) String() string { return "flap" }

func (static staticFlap) Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	assert := test.Assert{t}

	for _, r := range static.Routers {
		for _, i := range r.Intfs {
			var intf string
			if i.Vlan != "" {
				intf = i.Name + "." + i.Vlan
			} else {
				intf = i.Name
			}
			_, err := static.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "down", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			_, err = static.ExecCmd(t, r.Hostname,
				"ip", "link", "set", "up", intf)
			assert.Nil(err)
			time.Sleep(1 * time.Second)
			assert.Program(*Goes, "vnet", "show", "ip", "fib")
		}
	}
}

type staticInterConnectivity2 struct{ *docker.Docket }

func (staticInterConnectivity2) String() string { return "inter-connectivity2" }

func (static staticInterConnectivity2) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "10.1.0.2"},
		{"RA-1", "10.1.0.1"},
		{"RA-1", "10.2.0.3"},
		{"RA-1", "192.168.0.1"},
		{"RA-2", "10.2.0.2"},
		{"RA-2", "10.3.0.4"},
		{"RA-2", "192.168.0.2"},
		{"CA-2", "10.3.0.3"},
		{"CA-1", "10.3.0.4"},
		{"CA-1", "192.168.0.2"},
		{"CA-2", "10.1.0.1"},
		{"CA-2", "192.168.0.1"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(static.PingCmd(t, x.hostname, x.target))
		assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
			x.hostname)
	}
}

type staticPuntStress struct{ *docker.Docket }

func (staticPuntStress) String() string { return "punt-stress" }

func (static staticPuntStress) Test(t *testing.T) {
	if testing.Short() || *test.DryRun {
		t.SkipNow()
	}

	assert := test.Assert{t}
	assert.Comment("Check punt stress with iperf3")

	done := make(chan bool, 1)

	go func(done chan bool) {
		static.ExecCmd(t, "CA-2", "timeout", "15", "iperf3", "-s")
		done <- true
	}(done)

	time.Sleep(1 * time.Second)
	out, err := static.ExecCmd(t, "CA-1", "iperf3", "-c", "10.3.0.4")

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
