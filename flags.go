// Copyright © 2015-2018 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import "flag"

var (
	IsAlpha = flag.Bool("test.alpha", false, "zero based ports")
	Goes    = flag.String("test.goes", "./goes-platina-mk1",
		"GO Embedded System for Platina's Mk1 TOR Switch")
	SingleStep = flag.Bool("test.step", false, "single step (manual testing)")
	NoVnet = flag.Bool("test.novnet", false, "manual vnet start (debugger)")
)
