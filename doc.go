/*
This provides a blackbox test of goes-platina-mk1.

	go build -buildmode=plugin github.com/platinasystems/fe1/fe1
	go build github.com/platinasystems/goes-platina-mk1
	go test -c
	for f in testdata/*.yaml; do
		editor $f
		git update-index --assume-unchanged $f
	done
	sudo ./goes-platina-mk1-blackbox.test -help
*/
package main
