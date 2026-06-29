# TODO — Shortly Before Release

Items deferred from the POC→production hardening pass. Address before GA.

## 1. Set production poll interval (currently 15s, spec says 5 min)

**File:** `sensor/common/virtualmachine/vmscraper/scraper.go`

Change `defaultPollInterval` from `15 * time.Second` to `5 * time.Minute`.
The 15s value is useful during development/testing — flip to production value
before cutting the release.

## ~~2. Reduce per-VM logging from Info to Debug~~ ✓ Done

Per-VM lines (`Debugf`), cycle summary (`Infof`), errors (`Warnf`).
