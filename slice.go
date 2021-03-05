// Copyright © 2015-2018 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/docker"
)

func sliceVlanTest(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	sliceTest(t, "testdata/net/slice/vlan/conf.yaml.tmpl")
}

func sliceTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		sliceConnectivity{docket},
		sliceFrr{docket},
		sliceRoutes{docket},
		sliceInterConnectivity{docket},
		sliceIsolation{docket},
		sliceStress{docket},
		sliceConnectivity{docket},
		sliceRoutes{docket},
		sliceInterConnectivity{docket},
		sliceStressPci{docket},
		sliceConnectivity{docket},
		sliceRoutes{docket},
		sliceInterConnectivity{docket})
}

type sliceConnectivity struct{ *docker.Docket }

func (sliceConnectivity) String() string { return "connectivity" }

func (slice sliceConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "10.1.0.2"},
		{"RA-1", "10.1.0.1"},
		{"RA-1", "10.2.0.3"},
		{"RA-2", "10.2.0.2"},
		{"RA-2", "10.3.0.4"},
		{"CA-2", "10.3.0.3"},
		{"CB-1", "10.1.0.2"},
		{"RB-1", "10.1.0.1"},
		{"RB-1", "10.2.0.3"},
		{"RB-2", "10.2.0.2"},
		{"RB-2", "10.3.0.4"},
		{"CB-2", "10.3.0.3"},
	} {
		assert.Nil(slice.PingCmd(t, x.hostname, x.target))
		//FIXME
		//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
		//	x.hostname)
	}
}

type sliceFrr struct{ *docker.Docket }

func (sliceFrr) String() string { return "frr" }

func (slice sliceFrr) Test(t *testing.T) {
	assert := test.Assert{t}
	time.Sleep(1 * time.Second)
	for _, r := range slice.Routers {
		assert.Comment("Checking FRR on", r.Hostname)
		out, err := slice.ExecCmd(t, r.Hostname, "ps", "ax")
		assert.Nil(err)
		assert.True(regexp.MustCompile(".*ospfd.*").MatchString(out))
		assert.True(regexp.MustCompile(".*zebra.*").MatchString(out))
	}
}

type sliceRoutes struct{ *docker.Docket }

func (sliceRoutes) String() string { return "routes" }

func (slice sliceRoutes) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		route    string
	}{
		{"CA-1", "10.3.0.0/24"},
		{"CA-2", "10.1.0.0/24"},
		{"CB-1", "10.3.0.0/24"},
		{"CB-2", "10.1.0.0/24"},
	} {
		found := false
		timeout := 120
		for i := timeout; i > 0; i-- {
			out, err := slice.ExecCmd(t, x.hostname,
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

type sliceInterConnectivity struct{ *docker.Docket }

func (sliceInterConnectivity) String() string { return "inter-connectivity" }

func (slice sliceInterConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"CA-1", "10.3.0.4"}, // In slice A ping from CA-1 to CA-2
		{"CB-1", "10.3.0.4"}, // In slice B ping from CB-1 to CB-2
		{"CA-2", "10.1.0.1"}, // In slice A ping from CA-2 to CA-1
		{"CB-2", "10.1.0.1"}, // In slice B ping from CB-2 to CB-1

	} {
		assert.Nil(slice.PingCmd(t, x.hostname, x.target))
		//FIXME
		//assert.Program(*Goes, "vnet", "show", "ip", "fib", "table",
		//	x.hostname)
	}
}

type sliceIsolation struct{ *docker.Docket }

func (sliceIsolation) String() string { return "isolation" }

func (slice sliceIsolation) Test(t *testing.T) {
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
	_, err = slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
	assert.NonNil(err)

	assert.Comment("Verify that slice A is not affected")
	_, err = slice.ExecCmd(t, "CA-1", "ping", "-c1", "10.3.0.4")
	assert.Nil(err)
	//FIXME
	//assert.Program(regexp.MustCompile("10.3.0.0/24"),
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
	_, err = slice.ExecCmd(t, "CA-1", "ping", "-c1", "10.3.0.4")
	assert.NonNil(err)

	ok := false
	assert.Comment("Verify that slice B is not affected")
	timeout := 120
	for i := timeout; i > 0; i-- {
		out, _ := slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
		if !assert.MatchNonFatal(out, "1 received") {
			time.Sleep(1 * time.Second)
		} else {
			ok = true
			break
		}
	}
	if !ok {
		t.Error("Slice B ping failed")
	}
	//FIXME
	//assert.Program(regexp.MustCompile("10.3.0.0/24"),
	//	*Goes, "vnet", "show", "ip", "fib", "table", "RB-2")

	// bring RA-1 interfaces back up
	for _, i := range r.Intfs {
		intf := docker.IntfVlanName(i.Name, i.Vlan)
		_, err := slice.ExecCmd(t, r.Hostname,
			"ip", "link", "set", "up", intf)
		assert.Nil(err)
	}

}

func getCpuTemp() (val string, err error) {
	var (
		out    []byte
		re     *regexp.Regexp
		result []string
	)

	out, _ = exec.Command(*Goes, "hget", "platina-mk1", "temp").Output()
	re, _ = regexp.Compile(`sys.cpu.coretemp.C:\s+(\d+)`)
	result = re.FindStringSubmatch(string(out))
	if len(result) == 2 {
		val = result[1]
	} else {
		err = fmt.Errorf("temp regex failed [%v]\n", string(out))
	}
	return
}

func getIpv6Ll() (val string, err error) {
	var (
		out    []byte
		re     *regexp.Regexp
		result []string
	)

	out, _ = exec.Command(*Goes, "mac-ll").Output()
	re, _ = regexp.Compile(`IPv6 link-local:\s+([a-f0-9:]+)`)
	result = re.FindStringSubmatch(string(out))
	if len(result) == 2 {
		val = result[1] + "%eth0" // TODO fix non eth0 management
	} else {
		err = fmt.Errorf("mac-ll regex failed [%v]\n", string(out))
	}

	return
}

func getFanRpm() (rpm string, err error) {
	var (
		out     []byte
		lladdr6 string
	)

	lladdr6, _ = getIpv6Ll()

	out, _ = exec.Command("redis-cli", "--raw", "-h", lladdr6, "hget", "platina-mk1-bmc",
		"fan_tray.1.1.speed.units.rpm").Output()
	rpm = string(out)
	rpm = strings.TrimSuffix(rpm, "\n")

	return
}

type sliceStress struct{ *docker.Docket }

func (sliceStress) String() string { return "stress" }

func (slice sliceStress) Test(t *testing.T) {
	assert := test.Assert{t}

	assert.Comment("stress with hping3")

	// 1st check temp and fan speed
	// and compare at the end
	var (
		temp [6]string
		rpm  [6]string
		err  error
	)

	temp[0], err = getCpuTemp()
	assert.Nil(err)
	rpm[0], err = getFanRpm()
	assert.Nil(err)

	duration := []string{"1", "10", "30", "60"}

	ok := false
	timeout := 120
	for i := timeout; i > 0; i-- {
		out, _ := slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
		if !assert.MatchNonFatal(out, "1 received") {
			time.Sleep(1 * time.Second)
		} else {
			ok = true
			assert.Comment("ping ok before stress")
			break
		}
	}
	if !ok {
		t.Error("ping failing before stress test")
	}

	for i, to := range duration {
		assert.Comment("stress for", to)
		_, err := slice.ExecCmd(t, "CB-1",
			"timeout", "-s", "KILL", to,
			"hping3", "--icmp", "--flood", "-q", "10.3.0.4")
		assert.Comment("verfy can still ping neighbor")
		_, err = slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.1.0.2")
		if err != nil {
			assert.Comment("hping3 failed ", to)
		}
		assert.Nil(err)

		temp[i+1], err = getCpuTemp()
		assert.Nil(err)
		rpm[i+1], err = getFanRpm()
		assert.Nil(err)
	}

	temp[5], err = getCpuTemp()
	assert.Nil(err)
	rpm[5], err = getFanRpm()
	assert.Nil(err)

	assert.Commentf("Temp %vC, Fan %v before stress\n", temp[0], rpm[0])
	for i := 0; i <= 3; i++ {
		assert.Commentf("Temp %vC, Fan %v after %v stress\n", temp[i], rpm[i], duration[i])
	}
	assert.Commentf("Temp %vC, Fan %v after stress\n", temp[5], rpm[5])
}

type sliceStressPci struct{ *docker.Docket }

func (sliceStressPci) String() string { return "stress-pci" }

func (slice sliceStressPci) Test(t *testing.T) {
	assert := test.Assert{t}

	assert.Comment("stress with hping3 with ttl=1")

	duration := []string{"1", "10", "30", "60"}

	ok := false
	timeout := 120
	for i := timeout; i > 0; i-- {
		out, _ := slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.3.0.4")
		if !assert.MatchNonFatal(out, "1 received") {
			time.Sleep(1 * time.Second)
		} else {
			ok = true
			assert.Comment("ping ok before stress")
			break
		}
	}
	if !ok {
		t.Error("ping failing before stress test")
	}

	for _, to := range duration {
		assert.Comment("stress for", to)
		_, err := slice.ExecCmd(t, "CB-1",
			"timeout", "-s", "KILL", to,
			"hping3", "--icmp", "--flood", "-q", "-t", "1",
			"10.3.0.4")
		assert.Comment("verfy can still ping neighbor")
		_, err = slice.ExecCmd(t, "CB-1", "ping", "-c1", "10.1.0.2")
		if err != nil {
			assert.Comment("hping3 failed ", to)
		}

		assert.Nil(err)
	}
}
