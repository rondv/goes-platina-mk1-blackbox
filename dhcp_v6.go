// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/docker"
)

func dhcpNetV6Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	dhcpV6Test(t, "testdata/net6/dhcp/conf.yaml.tmpl")
}

func dhcpVlanV6Test(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	dhcpV6Test(t, "testdata/net6/dhcp/vlan/conf.yaml.tmpl")
}

func dhcpV6Test(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		dhcpV6Connectivity{docket},
		dhcpV6Server{docket},
		dhcpV6Client{docket},
		dhcpV6Connectivity2{docket},
		dhcpV6VlanTag{docket})
}

type dhcpV6Connectivity struct{ *docker.Docket }

func (dhcpV6Connectivity) String() string { return "connectivity" }

func (dhcp dhcpV6Connectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		host   string
		target string
	}{
		{"R1", "2001:db8:0:120::10"},
		{"R2", "2001:db8:0:120::5"},
	} {
		assert.Nil(dhcp.PingCmd(t, x.host, x.target))
		//FIXME
		//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
		//	x.host)
	}

	// enable dhcpd for IPv6
	_, err := dhcp.ExecCmd(t, "R2", "supervisorctl", "start", "dhcpd6")
	assert.Nil(err)
}

type dhcpV6Server struct{ *docker.Docket }

func (dhcpV6Server) String() string { return "server" }

func (dhcp dhcpV6Server) Test(t *testing.T) {
	assert := test.Assert{t}

	assert.Comment("Checking dhcp server on", "R2")
	time.Sleep(1 * time.Second)
	out, err := dhcp.ExecCmd(t, "R2", "ps", "ax")
	assert.Nil(err)
	//assert.Match(out, ".*dhcpd.*")
	timeout := 5
	found := false
	for i := timeout; i > 0; i-- {
		if !assert.MatchNonFatal(out, ".*dhcpd.*") {
			if *test.VV {
				fmt.Printf("check R2 ps ax, no match on dhcpd, %v retries left\n", i-1)
				fmt.Printf("%v\n", out)
			}
			time.Sleep(2 * time.Second)
			out, err = dhcp.ExecCmd(t, "R2", "ps", "ax")
			continue
		}
		found = true
	}
	if !found {
		test.Pause.Prompt("dhcpd not found")
		assert.Nil(fmt.Errorf("check dhcpd failed\n"))
	}
}

type dhcpV6Client struct{ *docker.Docket }

func (dhcpV6Client) String() string { return "client" }

func (dhcp dhcpV6Client) Test(t *testing.T) {
	assert := test.Assert{t}

	r, err := docker.FindHost(dhcp.Config, "R1")
	intf := r.Intfs[0]

	// remove existing IP address
	_, err = dhcp.ExecCmd(t, "R1",
		"ip", "address", "delete", "2001:db8:0:120::5/64", "dev", intf.Name)
	assert.Nil(err)

	assert.Comment("Verify ping fails")
	_, err = dhcp.ExecCmd(t, "R1", "ping", "-c1", "2001:db8:0:120::10")
	assert.NonNil(err)

	assert.Comment("Request dhcp address")
	out, err := dhcp.ExecCmd(t, "R1", "dhclient", "-6", "-v", intf.Name)
	assert.Nil(err)
	assert.Match(out, "Bound to")

	// dhcpv6 does gives a /128 and no default route
	_, err = dhcp.ExecCmd(t, "R1", "ip", "-6", "route", "add", "2001:db8:0:120::10/128", "dev", intf.Name)
	assert.Nil(err)
	_, err = dhcp.ExecCmd(t, "R1", "ip", "-6", "route", "add", "2001:db8:0:120::/64", "via", "2001:db8:0:120::10")
	assert.Nil(err)
}

type dhcpV6Connectivity2 struct{ *docker.Docket }

func (dhcpV6Connectivity2) String() string { return "connectivity2" }

func (dhcp dhcpV6Connectivity2) Test(t *testing.T) {
	assert := test.Assert{t}

	assert.Comment("Check connectivity with dhcp address")
	assert.Nil(dhcp.PingCmd(t, "R1", "2001:db8:0:120::10"))
	//FIXME
	//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table", "R1")
	//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table", "R2")
}

type dhcpV6VlanTag struct{ *docker.Docket }

func (dhcpV6VlanTag) String() string { return "vlanTag" }

func (dhcp dhcpV6VlanTag) Test(t *testing.T) {
	assert := test.Assert{t}

	assert.Comment("Check for invalid vlan tag") // issue #92

	r1, err := docker.FindHost(dhcp.Config, "R1")
	r1Intf := r1.Intfs[0]

	// remove existing IP address
	_, err = dhcp.ExecCmd(t, "R1",
		"ip", "-6", "address", "flush", "dev", r1Intf.Name)
	assert.Nil(err)

	// flush also removes the link local, flap interface to get it back
	_, err = dhcp.ExecCmd(t, "R1",
		"ip", "link", "set", "down", r1Intf.Name)
	assert.Nil(err)
	_, err = dhcp.ExecCmd(t, "R1",
		"ip", "link", "set", "up", r1Intf.Name)
	assert.Nil(err)
	time.Sleep(1 * time.Second)

	r2, err := docker.FindHost(dhcp.Config, "R2")
	r2Intf := r2.Intfs[0]

	_, err = dhcp.ExecCmd(t, "R1", "pkill", "dhclient")

	done := make(chan bool, 1)
	go func(done chan bool) {
		out, err := dhcp.ExecCmd(t, "R2",
			"timeout", "10",
			"tcpdump", "-c1", "-nvvvei", r2Intf.Name, "port", "547")
		assert.Nil(err)
		match, err := regexp.MatchString("vlan 0", out)
		assert.Nil(err)
		if match {
			t.Error("Invalid vlan 0 tag found")
		}
		done <- true
	}(done)

	time.Sleep(1 * time.Second)
	_, err = dhcp.ExecCmd(t, "R1", "dhclient", "-6", "-v", r1Intf.Name)
	assert.Nil(err)
	<-done
}
