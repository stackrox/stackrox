# flog
[![Build Status](https://travis-ci.org/facebookincubator/flog.svg?branch=master)](https://travis-ci.org/facebookincubator/flog)
[![GoDoc](https://godoc.org/github.com/facebookincubator/flog?status.svg)](https://godoc.org/github.com/facebookincubator/flog)

Flog is a hacked and slashed version of glog that can be used as a drop in
replacement for the latter.

## Overview
Flog was originally created to facilitate logging from within libraries but can be used in main programs as well.

To this end, this library has some significant differences, compared to the
original glog. Namely:

* It only logs to stderr. Logging to files, along with all relevant flags and
tests has been removed.
* It supports configuration through env vars as well as a configuration struct
that allows for more flexibility when using the lib.
* Users can log immediately without first having to call flag.Parse().
* It only adds flags for the lib explicitly.
* It has a different buffer allocation mechanism that works faster in scenarios
where parallel logging is required (see the BenchmarkHeaderParallel).
* Support to get the current verbosity level.
* Support to set a different output writer.
* Two more severity levels added, DEBUG and CRITICAL, along with their relevant
Debug*() and Critical*() functions.

However, the important parts of glog have been retained, such as:

* The V() functionality remains the same.
* Users can use Info*(), Warning*(), Error*() and Fatal*() functions as before.
* Filtering through module name and tracebacks are supported in exactly the
same way as in glog.

## Configuration

As noted above configuration is possible through different ways. We outline them
below.

### Environment Variables

flog supports 3 different env vars for configuring its behavior. These are:

* FLOG_VERBOSITY - takes an int > 0 argument and will set the overall Verbosity
for the lib.
* FLOG_VMODULE - takes a string argument containing a pattern which is then used
to filter logs from different module thus allowing to setup different
verbosities for different parts of the program.
* FLOG_LOG_BACKTRACE_AT - takes a string argument so that when logging from a
particular line in a particular file a stack trace is also printed.

These vars are considered during package initialization through its init()
function.

### CLI flags

As with the original glog, flog also supports adding flags that configure the
behavior described above. The flags are -v, -vmodule and -log_backtrace_at and
their meaning is equivalent to the env vars described above.
Unlike glog however, these flags are added only after an explicit call to the
AddFlags() function of the package and only support the flag Go package. This
call will add all flags to a given flag set. The second argument is a config
structure (cf. Section 2.3) which, if not nil, will have its Set() method called
before setting any flags. This function returns either any errors produced by
Set() or nil.

### The Config structure

This is a flexible interface that we've added to allow programs to create more
complex logging configurations where parts of the program may log with different
verbosities and can request different filters.

The struct contains the following members, Verbosity, Vmodule and TraceLocation
and their meaning is the same as the flags described above. All the members of
this struct are strings.
The caller must call the Set() method of this struct to set the values. This
method is concurrency-safe.

### Precedence

The order by which this lib honors the above configuration options is:

1. Environment Variables
2. Flags
3. Any values set through Config objects

### Defaults

The default values are as follows:

* Verbosity = 0
* Vmodule = ""
* Log Backtrace At = ""

## License
Flog is published under the Apache v2.0 License.
