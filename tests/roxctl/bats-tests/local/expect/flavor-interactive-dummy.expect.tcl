#!/usr/bin/expect -f
# Syntax of this file is called TCL

# Some knowledge about expect:
# https://www.pantz.org/software/expect/expect_examples_and_tips.html
# https://man7.org/linux/man-pages/man1/expect.1.html
# O'reilly Book about Expect: https://www.oreilly.com/library/view/exploring-expect/9781565920903/

# This test can be run locally with:
# expect -f "tests/roxctl/bats-tests/local/expect/flavor-interactive.expect.tcl" -- <path-to-roxctl> <flavor-name> "$(mktemp -d -u)" <expected-prefix-of-image-registry-in-prompt>

# exp_internal 1 # uncomment for debug mode
# wait at most 10 seconds for a question to appear - applies for each question
set timeout 10
set binary [lindex $argv 0]
set out_dir [lindex $argv 1]


if {[llength $argv] != 2} {
  send_user "Usage: expect <script> <out_dir>\n"
  exit 1
}

spawn {*}"$binary" central generate interactive

expect "Enter path to the backup bundle from which to restore keys and certificates*: " { send "\n" }
expect "Enter read templates from local filesystem*:*: " { send "\n" }
expect "Enter path to helm templates on your local filesystem*:*: " { send "\n" }
expect "Enter PEM cert bundle file*: " { send "\n" }
expect "Enter Create PodSecurityPolicy resources*:*: " { send "\n" }
expect "Enter administrator password*:*: " { send "\n" }
expect "Enter orchestrator (k8s, openshift)*: " { send "k8s\n" }
# Sending invalid value
expect "Enter default container images settings*:*:" { send "dummy\n" }

expect {
  "Unexpected value 'dummy', allowed values are*" {
    send "rhacs\n"
    # ensure that the next question is correct after providing a valid answer
    expect "Enter the directory to output the deployment bundle to*" {
      exit 0
    }
    send_user "\nERROR: roxctl accepted 'rhacs' as flavor and generated unexpected question afterwards\n"
    exit 2
  }
  "Enter the directory to output the deployment bundle to*" {
    send_user "\nERROR: roxctl accepted 'dummy' as flavor and did not ask for correction immediately\n"
    exit 1
  }
}
