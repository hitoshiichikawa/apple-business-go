// Package livetest holds a build-tagged, real-tenant verification test for the
// Apple Business API. The test file (live_test.go) requires the `livetest`
// build tag, so ordinary `go build ./...` and `go test ./...` ignore it and do
// not hit the network.
//
// This file carries no build tag on purpose: it keeps the package non-empty so
// that `./...` commands do not fail on a directory whose only Go file is
// excluded by a build constraint.
//
// To run the verification against a real tenant, see README.md.
package livetest
