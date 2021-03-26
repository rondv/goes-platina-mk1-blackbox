// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"testing"

	"github.com/platinasystems/test"
	"github.com/platinasystems/test/docker"
)

func routesNetTest(t *testing.T) {
	routesTest(t, "testdata/routes/conf.yaml.tmpl")
}

func routesVlanTest(t *testing.T) {
	routesTest(t, "testdata/routes/vlan/conf.yaml.tmpl")
}

func routesTest(t *testing.T, tmpl string) {
	docket := &docker.Docket{Tmpl: tmpl}
	docket.Test(t,
		routesConnectivity{docket},
		routesAdd900{docket},
		routesConnectivity{docket},
		routesDel900{docket},
		routesConnectivity{docket},
		routesAdd1500{docket},
		routesConnectivity{docket},
		routesDel1500{docket},
		routesConnectivity{docket},
		routesAdd4500{docket},
		routesConnectivity{docket},
		routesDel4500{docket},
		routesConnectivity{docket},
	)
}

type routesConnectivity struct{ *docker.Docket }

func (routesConnectivity) String() string { return "connectivity" }

func (routes routesConnectivity) Test(t *testing.T) {
	assert := test.Assert{t}

	for _, x := range []struct {
		hostname string
		target   string
	}{
		{"H1", "10.1.0.1"},
		{"R1", "192.168.1.2"},
		{"R1", "10.1.0.2"},
		{"R1", "10.2.0.2"},
		{"R1", "192.168.2.2"},
		{"H2", "10.2.0.1"},

		{"H1", "2001:db8:1::1"},
		{"R1", "2001:db8:0:1::2"},
		{"R1", "2001:db8:1::2"},
		{"R1", "2001:db8:2::2"},
		{"R1", "2001:db8:0:2::2"},
		{"H2", "2001:db8:2::1"},

		{"H1", "10.2.0.1"},
		{"H1", "10.2.0.2"},
		{"H1", "192.168.2.2"},

		{"H1", "2001:db8:2::1"},
		{"H1", "2001:db8:2::2"},
		{"H1", "2001:db8:0:2::2"},
	} {
		assert.Comment("ping from", x.hostname, "to", x.target)
		assert.Nil(routes.PingCmd(t, x.hostname, x.target))
	}
}

type routesAdd900 struct{ *docker.Docket }

func (routesAdd900) String() string { return "add 900" }

func (routes routesAdd900) Test(t *testing.T) {
	assert := test.Assert{t}

	fileName := "/etc/frr/add900"

	_, err := routes.ExecCmd(t, "R1", "ip", "-b", fileName)
	assert.Nil(err)
}

type routesDel900 struct{ *docker.Docket }

func (routesDel900) String() string { return "del 900" }

func (routes routesDel900) Test(t *testing.T) {
	assert := test.Assert{t}

	fileName := "/etc/frr/del900"

	_, err := routes.ExecCmd(t, "R1", "ip", "-b", fileName)
	assert.Nil(err)
}

type routesAdd1500 struct{ *docker.Docket }

func (routesAdd1500) String() string { return "add 1500" }

func (routes routesAdd1500) Test(t *testing.T) {
	assert := test.Assert{t}

	fileName := "/etc/frr/add1500"

	_, err := routes.ExecCmd(t, "R1", "ip", "-b", fileName)
	assert.Nil(err)
}

type routesDel1500 struct{ *docker.Docket }

func (routesDel1500) String() string { return "del 1500" }

func (routes routesDel1500) Test(t *testing.T) {
	assert := test.Assert{t}

	fileName := "/etc/frr/del1500"

	_, err := routes.ExecCmd(t, "R1", "ip", "-b", fileName)
	assert.Nil(err)
}

type routesAdd4500 struct{ *docker.Docket }

func (routesAdd4500) String() string { return "add 4500" }

func (routes routesAdd4500) Test(t *testing.T) {
	assert := test.Assert{t}

	fileName := "/etc/frr/add4500"

	_, err := routes.ExecCmd(t, "R1", "ip", "-b", fileName)
	assert.Nil(err)
}

type routesDel4500 struct{ *docker.Docket }

func (routesDel4500) String() string { return "del 4500" }

func (routes routesDel4500) Test(t *testing.T) {
	assert := test.Assert{t}

	fileName := "/etc/frr/del4500"

	_, err := routes.ExecCmd(t, "R1", "ip", "-b", fileName)
	assert.Nil(err)
}
