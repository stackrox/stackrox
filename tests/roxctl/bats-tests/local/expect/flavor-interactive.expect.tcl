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
set flavor [lindex $argv 1]
set out_dir [lindex $argv 2]
set registry [lindex $argv 3]


# exitWith is an integer interpreted as binary field, where each bit denotes a different error
# 8 - a question has not been displayed
# 4 - prompt for entering main-image       is missing default value hint
# 2 - prompt for entering scanner-db-image is missing default value hint
# 1 - prompt for entering scanner-image    is missing default value hint
# 0 - no error
set exitWith 0

if {[llength $argv] != 4} {
  send_user "Usage: expect <script> <binary> <out_dir> <expected_registry_prefix>\n"
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
expect "Enter default container images settings*:*: " { send "$flavor\n" }
expect "Enter the directory to output the deployment bundle to*:*: " { send "$out_dir\n" }
expect "Enter whether to enable telemetry*:" { send "\n" }

# Enter central-db image to use (default: "docker.io/stackrox/central-db:2.21.0-15-g448f2dc8fa"):
# Enter central-db image to use (default: "stackrox.io/central-db:3.67.x-296-g56df6a892d"):
# Enter central-db image to use (default: "registry.redhat.io/advanced-cluster-security/rhacs-central-db-rhel8:3.68.x-30-g516b4e7a6c-dirty"):
expect {
  default {
    send_user "\nFATAL: No question about Central-DB image\n"
    exit 8
  }
  "Enter central-db * (if unset, the default will be used):" {
    send_user "WARNING: roxctl does not suggest any registry for central-db"
    send "\n"
    set exitWith [expr {$exitWith + 2}]
  }
  "Enter central-db * (default: \"$registry/central-db:*\"):" {
    send_user "roxctl suggests correct registry for central-db"
    send "\n"
  }
  # Special case for RHACS to avoid writing a regexp in TCL
  "Enter central-db * (default: \"$registry/rhacs-central-db-rhel8:*\"):" {
    send_user "roxctl suggests correct registry for central-db"
    send "\n"
  }
}

expect "Enter list of secrets to add as declarative configuration*:" { send "\n" }
expect "Enter the method of exposing Central*:*: " { send "none\n" }

# Enter main image to use (default: "docker.io/stackrox/main:3.67.x-296-g56df6a892d"):
# Enter main image to use (default: "stackrox.io/main:3.67.x-296-g56df6a892d")
# Enter main image to use (default: "registry.redhat.io/advanced-cluster-security/rhacs-main-rhel8:3.68.x-30-g516b4e7a6c-dirty"):
expect {
  default {
    send_user "\nFATAL: No question about main image\n"
    exit 8
  }
  "Enter main * (if unset, the default will be used):" {
    send_user "WARNING: roxctl does not suggest any registry for main"
    send "\n"
    set exitWith [expr {$exitWith + 4}]
  }
  "Enter main * (default: \"$registry/main:*\"):" {
    send_user "roxctl suggests correct registry for main"
    send "\n"
  }
  # Special case for RHACS to avoid writing a regexp in TCL
  "Enter main * (default: \"$registry/rhacs-main-rhel8:*\"):" {
    send_user "roxctl suggests correct registry for main"
    send "\n"
  }
}
expect "Enter whether to run StackRox in offline mode, which avoids reaching out to the Internet*" { send "\n" }
expect "Enter list of config maps to add as declarative configuration*:" { send "\n" }
expect "Enter the deployment tool to use (kubectl, helm, helm-values)*:" { send "\n" }
expect "Enter Istio version when deploying into an Istio-enabled cluster*:" { send "\n" }

# Enter scanner-db image to use (default: "docker.io/stackrox/scanner-db:2.21.0-15-g448f2dc8fa"):
# Enter scanner-db image to use (default: "stackrox.io/scanner-db:3.67.x-296-g56df6a892d"):
# Enter scanner-db image to use (default: "registry.redhat.io/advanced-cluster-security/rhacs-scanner-db-rhel8:3.68.x-30-g516b4e7a6c-dirty"):
expect {
  default {
    send_user "\nFATAL: No question about Scanner-DB image\n"
    exit 8
  }
  "Enter scanner-db * (if unset, the default will be used):" {
    send_user "WARNING: roxctl does not suggest any registry for scanner-db"
    send "\n"
    set exitWith [expr {$exitWith + 2}]
  }
  "Enter scanner-db * (default: \"$registry/scanner-db:*\"):" {
    send_user "roxctl suggests correct registry for scanner-db"
    send "\n"
  }
  # Special case for RHACS to avoid writing a regexp in TCL
  "Enter scanner-db * (default: \"$registry/rhacs-scanner-db-rhel8:*\"):" {
    send_user "roxctl suggests correct registry for scanner-db"
    send "\n"
  }
}
# Enter Scanner image to use (default: "docker.io/stackrox/scanner:2.21.0-15-g448f2dc8fa"):
# Enter Scanner image to use (default: "stackrox.io/scanner:3.67.x-296-g56df6a892d"):
expect {
  default {
    send_user "\nFATAL: No question about scanner image\n"
    exit 8
  }
  "Enter scanner * (if unset, the default will be used):" {
    send_user "exitWith before $exitWith"
    send_user "WARNING: roxctl does not suggest any registry for scanner"
    send "\n"
    set exitWith [expr {$exitWith + 1}]
    send_user "exitWith is now $exitWith"
  }
  "Enter scanner * (default: \"$registry/scanner:*\"):" {
    send_user "roxctl suggests correct registry for scanner"
    send "\n"
  }
  # Special case for RHACS to avoid writing a regexp in TCL
  "Enter scanner * (default: \"$registry/rhacs-scanner-rhel8:*\"):" {
    send_user "roxctl suggests correct registry for scanner"
    send "\n"
  }
}
expect "Enter Central volume type*:" { send "pvc\n" }
expect "Enter external volume name for Central*:" { send "\n" }
expect "Enter external volume size in Gi for Central*:" { send "\n" }
expect "Enter storage class name for Central (optional if you have a default StorageClass configured):" { send "\n" }

expect "Enter external volume name for Central DB*:" { send "\n" }
expect "Enter external volume size in Gi for Central DB*:" { send "\n" }
expect "Enter storage class name for Central DB (optional if you have a default StorageClass configured):" { send "\n" }

# Setting a generous timeout, as generating files may take >3 seconds
expect -timeout 20 "Generating deployment bundle..."
expect -timeout 20 "Wrote central bundle to \"$out_dir\""
expect -timeout 20 eof
exit "$exitWith"
