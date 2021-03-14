# pault.ag/go/diskring

[![Go Reference](https://pkg.go.dev/badge/pault.ag/go/diskring.svg)](https://pkg.go.dev/pault.ag/go/diskring)
[![Go Report Card](https://goreportcard.com/badge/pault.ag/go/diskring)](https://goreportcard.com/report/pault.ag/go/diskring)

pault.ag/go/diskring contains an implementation of a ring buffer, backed by a
file on the filesystem. This allows concurrent reads and writes by goroutines,
overwriting data as needed, oldest-first.

This buffer does not have fixed sizes, rather, it enocdes the length with
the written data. Data is added and removed at the chunk level.
