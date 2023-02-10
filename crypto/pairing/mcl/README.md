go-mcl
======

This is a wrapper for `herumi/mcl`'s elliptic curve library that you can import easily into your projects via a simple:

```
    import "github.com/alinush/go-mcl"
```

## Limitations

I disabled support for linking to the `bn256` and `bn384` libraries in `src/mcl.go`, so some curve benchmarks have been removed.
However, paradoxically, the `bn256` benchmarks still work.

## Documentation

You can see the documentation by running the doc server:

    ./scripts/run-doc-server.sh

Then, just browse to [http://localhost:6060/pkg/#thirdparty](http://localhost:6060/pkg/#thirdparty).

## Dependencies

This wrapper assumes `herumi/mcl` (or `alinush/mcl`) is installed locally.

Run the following script to take care of this and other dependencies:

    ./scripts/install-deps.sh

## Tests and benchmarks

First, set up the Go environment:

    source scripts/set-env.sh

To run all tests:

    ./scripts/test-mcl.sh

To run all benchmarks:

    ./scripts/bench-mcl.sh

## Differences from herumi/mcl's Golang bindings

Our `mcl.go` file is a combination of two files from [herumi/mcl](https://github.com/herumi/mcl):

    1. [ffi/go/mcl/init.go](https://github.com/herumi/mcl/blob/master/ffi/go/mcl/init.go)
        - This file just had an `Init()` function, so any future changes to it are easy to integrate.
    1. [ffi/go/mcl/mcl.go](https://github.com/herumi/mcl/blob/master/ffi/go/mcl/mcl.go)
        - For this file, a manual diff will need to be done between herumi's version and our version to integrate changes

So to integrate new [herumi/mcl](https://github.com/herumi/mcl) changes into this library, look at the differences via a:

    git clone https://github.com/herumi/mcl
    diff -rupN mcl/ffi/go/mcl/init.go init.go
    diff -rupN mcl/ffi/go/mcl/mcl.go  mcl.go

Pay close attention to changes in the `#cgo` directives in both of these files, since they will have to be carefully integrated if they change.
