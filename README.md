# percpu

[![Go Reference](https://pkg.go.dev/badge/github.com/martin-sucha/percpu.svg)](https://pkg.go.dev/github.com/martin-sucha/percpu)

Percpu is a Go package to support best-effort CPU-local sharded values.

This package is something of an experiment. See [Go issue #18802] for discussion
about adding this functionality into the Go standard library.

This version is updated to use generics.

## IMPORTANT CAVEATS

* This package uses `go:linkname` to access unexported functions from inside the
  Go runtime. Those could be changed or removed in a future Go version, breaking
  this package.
* It may be tempting to use this package to solve problems for which there are
  better solutions that do not break key abstractions of the runtime.

See [When to use percpu](using.md) for a discussion about when this package may
or may not be appropriate.

[Go issue #18802]: https://github.com/golang/go/issues/18802
