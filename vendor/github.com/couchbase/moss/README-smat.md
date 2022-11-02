# Instructions for smat testing for moss

[smat](https://github.com/mschoch/smat) is a framework that provides
state machine assisted fuzz testing.

To run the smat tests for moss...

## Prerequisites

    $ go get github.com/dvyukov/go-fuzz/go-fuzz
    $ go get github.com/dvyukov/go-fuzz/go-fuzz-build

## Steps

1.  Generate initial smat corpus:
```
    go test -tags=gofuzz -run=TestGenerateSmatCorpus
```

2.  Build go-fuzz test program with instrumentation:
```
    go-fuzz-build github.com/couchbase/moss
```

3.  Run go-fuzz:
```
    go-fuzz -bin=./moss-fuzz.zip -workdir=workdir/ -timeout=2000
```
