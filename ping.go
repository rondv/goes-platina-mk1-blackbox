// Copyright © 2015-2018 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/netport"
)

var Flood = flag.Int("test.flood", 1, "flood ping duration in seconds")

func pingTest(t *testing.T, netdevs netport.NetDevs) {
	netdevs.Test(t,
		pingGateways(netdevs),
		pingRemotes(netdevs),
		pingFlood(netdevs),
	)
}

type pingGateways []netport.NetDev

func (pingGateways) String() string {
	return "gateways"
}

func (list pingGateways) Test(t *testing.T) {
	assert := test.Assert{t}
	for _, nd := range []netport.NetDev(list) {
		for _, r := range nd.Routes {
			assert.Ping(nd.Netns, r.GW)
		}
	}
}

type pingRemotes []netport.NetDev

func (pingRemotes) String() string {
	return "remotes"
}

func (list pingRemotes) Test(t *testing.T) {
	assert := test.Assert{t}
	for _, nd := range []netport.NetDev(list) {
		for _, r := range nd.Remotes {
			assert.Ping(nd.Netns, r)
		}
	}
}

type pingFlood []netport.NetDev

func (pingFlood) String() string {
	return fmt.Sprint("flood-", time.Duration(*Flood)*time.Second)
}

func (list pingFlood) Test(t *testing.T) {
	if *Flood <= 0 {
		t.SkipNow()
	}
	assert := test.Assert{t}
	nd := []netport.NetDev(list)[0]
	ns := nd.Netns
	gw := nd.Routes[0].GW
	dur := time.Duration(*Flood) * time.Second
	assert.Ping(ns, gw)
	p, err := test.Begin(t, dur, test.Quiet{},
		"ip", "netns", "exec", ns,
		"hping3", "--icmp", "--flood", "-q", "-t", 1, gw)
	assert.Nil(err)
	p.End()
	assert.Ping(ns, gw)
}
