git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    commit = "70fe781d940c7b9cfee164d421ea85451322ba50",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

load("@io_bazel_rules_go//proto:def.bzl", "proto_register_toolchains")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_grpc_library")

