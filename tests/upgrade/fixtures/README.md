# Upgrade test fixtures

`scale/launch_workload.sh` is an unmodified snapshot of the scale workload
script from commit `7b817fa511ac4533cdf2d79a1b8e04d4d7557ad2`. Its SHA-256 is
`af6e37464251292ed7e2c6463f1b0d3ae612c6a256d4fb4aafedba3626237f08`.

If `EARLIER_SHA` in `postgres_run.sh` changes, refresh this fixture from the
new pinned commit and update the CI scale patch and its tests together.
