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

const (
	AtVnetd = "@platina-mk1/vnetd"
	AtXeth  = "@xeth"
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
		ecode = m.Run()
		return
	}
	if os.Geteuid() != 0 {
		panic("you aren't root")
	}
	if b, err := ioutil.ReadFile("/proc/net/unix"); err == nil {
		if bytes.Index(b, []byte(AtVnetd)) >= 0 {
			panic(fmt.Errorf("%s %s", AtVnetd, "in use"))
		}
		if bytes.Index(b, []byte(AtXeth)) >= 0 {
			test.Run("rmmod", "platina-mk1")
		}
	}
	loadXeth()
	netport.Init()
	ethtool.Init()
	redisd.Start(*Goes, "redisd")
	defer redisd.Stop()
	test.Run(*Goes, "hwait", "platina-mk1", "redis.ready", "true", "10")
	vnetd.Start(*Goes, "vnetd")
	defer vnetd.Stop()
	test.Pause("attach vnet debugger to pid ", vnetd.Pid())
	test.Run(*Goes, "hwait", "platina-mk1", "vnet.ready", "true", "30")
	ecode = m.Run()
}

func Test(t *testing.T) {
	t.Run("net", func(t *testing.T) {
		t.Run("ping", func(t *testing.T) {
			pingTest(t, netport.TwoNets)
		})
		t.Run("static", func(t *testing.T) {
			staticTest(t, tfn("net/static"))
		})
		t.Run("gobgp", func(t *testing.T) {
			gobgpTest(t, tfn("gobgp/ebgp"))
		})
		t.Run("bird", func(t *testing.T) {
			t.Run("bgp", func(t *testing.T) {
				birdBgpTest(t, tfn("bird/bgp"))
			})
			t.Run("ospf", func(t *testing.T) {
				birdOspfTest(t, tfn("bird/ospf"))
			})
			if *test.DryRun {
				t.SkipNow()
			}
		})
		t.Run("frr", func(t *testing.T) {
			t.Run("bgp", func(t *testing.T) {
				frrBgpTest(t, tfn("frr/bgp"))
			})
			t.Run("ospf", func(t *testing.T) {
				frrOspfTest(t, tfn("frr/ospf"))
			})
			t.Run("isis", func(t *testing.T) {
				frrIsisTest(t, tfn("frr/isis"))
			})
			if *test.DryRun {
				t.SkipNow()
			}
		})
		if *test.DryRun {
			t.SkipNow()
		}
	})
	t.Run("vlan", func(t *testing.T) {
		t.Run("ping", func(t *testing.T) {
			pingTest(t, netport.TwoVlanNets)
		})
		t.Run("slice", func(t *testing.T) {
			sliceTest(t, tfn("net/slice/vlan"))
		})
		t.Run("static", func(t *testing.T) {
			staticTest(t, tfn("net/static/vlan"))
		})
		t.Run("gobgp", func(t *testing.T) {
			gobgpTest(t, tfn("gobgp/ebgp/vlan"))
		})
		t.Run("bird", func(t *testing.T) {
			t.Run("bgp", func(t *testing.T) {
				birdBgpTest(t, tfn("bird/bgp/vlan"))
			})
			t.Run("ospf", func(t *testing.T) {
				birdOspfTest(t, tfn("bird/ospf/vlan"))
			})
			if *test.DryRun {
				t.SkipNow()
			}
		})
		t.Run("frr", func(t *testing.T) {
			t.Run("bgp", func(t *testing.T) {
				frrBgpTest(t, tfn("frr/bgp/vlan"))
			})
			t.Run("ospf", func(t *testing.T) {
				frrOspfTest(t, tfn("frr/ospf/vlan"))
			})
			t.Run("isis", func(t *testing.T) {
				frrIsisTest(t, tfn("frr/isis/vlan"))
			})
			if *test.DryRun {
				t.SkipNow()
			}
		})
		if *test.DryRun {
			t.SkipNow()
		}
	})
	if *test.DryRun {
		t.SkipNow()
	}
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

func tfn(dir string) string {
	return "testdata/" + dir + "/conf.yaml.tmpl"
}
