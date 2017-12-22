load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["fixtures.go"],
    data = ["fixtures"],
    importpath = "bitbucket.org/stack-rox/apollo/image/policies",
    visibility = ["//visibility:public"],
    deps = ["//pkg/testutils:go_default_library"],
)

filegroup(
    name = "fixtures",
    srcs = glob(["image/*.json"]),
    visibility = ["//visibility:public"],
)
