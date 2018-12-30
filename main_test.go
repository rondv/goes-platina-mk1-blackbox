// Copyright © 2015-2018 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/ethtool"
	"github.com/platinasystems/test/netport"
)

func TestMain(m *testing.M) {
	var ecode int
	var redisd, vnetd test.Daemon
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, r)
			ecode = 1
		}
		if ecode != 0 {
			os.Exit(ecode)
		}
	}()
	flag.Parse()
	if *test.DryRun {
		m.Run()
		return
	}
	if os.Geteuid() != 0 {
		panic("you aren't root")
	}
	if b, err := ioutil.ReadFile("/proc/net/unix"); err == nil {
		for _, atsock := range []string{
			"@redisd",
			"@redis.reg",
			"@redis.pub",
			"@vnet",
			"@vnetd",
		} {
			if bytes.Index(b, []byte(atsock)) >= 0 {
				panic(fmt.Errorf("%s %s", atsock, "in use"))
			}
		}
		if bytes.Index(b, []byte("@xeth")) >= 0 {
			test.Run("rmmod", "platina-mk1")
		}
	}
	loadXeth()
	netport.Init()
	ethtool.Init()
	redisd.Start(*Goes, "redisd")
	defer redisd.Stop()
	test.Run(*Goes, "hwait", "platina-mk1", "redis.ready", "true", "10")
	if *NoVnet {
		test.Pause("run vnet-platina-mk1")
	} else {
		vnetd.Start(*Goes, "vnetd")
		defer vnetd.Stop()
	}
	test.Run(*Goes, "hwait", "platina-mk1", "vnet.ready", "true", "30")
	ecode = m.Run()
}

func Test(t *testing.T) {
	t.Run("net", func(t *testing.T) {
		t.Run("ping", pingNetTest)
		t.Run("dhcp", dhcpNetTest)
		t.Run("static", staticNetTest)
		t.Run("gobgp", gobgpNetTest)
		t.Run("bird", birdNetTest)
		t.Run("frr", frrNetTest)
		test.SkipIfDryRun(t)
	})
	t.Run("vlan", func(t *testing.T) {
		t.Run("ping", pingVlanTest)
		t.Run("dhcp", dhcpVlanTest)
		t.Run("slice", sliceVlanTest)
		t.Run("static", staticVlanTest)
		t.Run("gobgp", gobgpVlanTest)
		t.Run("bird", birdVlanTest)
		t.Run("frr", frrVlanTest)
		test.SkipIfDryRun(t)
	})
	t.Run("nsif", nsifTest)
	t.Run("multipath", mpTest)
	test.SkipIfDryRun(t)
}

func loadXeth() {
	const ko = "platina-mk1.ko"
	xargs := []string{"modprobe", "platina-mk1"}
	if _, err := os.Stat(ko); err == nil {
		xargs = []string{"insmod", ko}
	}
	if *IsAlpha {
		xargs = append(xargs, "alpha=1")
	}
	if *test.VVV {
		xargs = append(xargs, "dyndbg=+pmf")
	} else {
		xargs = append(xargs, "dyndbg=-pmf")
	}
	test.Run(xargs...)
}
