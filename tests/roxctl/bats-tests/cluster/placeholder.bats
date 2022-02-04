#!/usr/bin/env bats

load "../helpers.bash"

@test "placeholder" {
  run true
  assert_success
}
