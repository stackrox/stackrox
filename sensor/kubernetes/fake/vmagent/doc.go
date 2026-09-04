// Package vmagent is an in-process stand-in for roxagent used by fake VM workloads.
// Dial returns a nop stream; GetReport ignores it and returns a generated v4 report
// so VMScraper poll, spread, backoff, and forward run without VSOCK or TLS.
package vmagent
