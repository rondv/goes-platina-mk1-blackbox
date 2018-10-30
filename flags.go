// Copyright © 2015-2018 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import "flag"

var (
	IsAlpha = flag.Bool("test.alpha", false, "this is a zero based alpha system")
	Goes    = flag.String("test.goes", "./goes-platina-mk1",
		"GO Embedded System for Platina's Mk1 TOR Switch")
)
