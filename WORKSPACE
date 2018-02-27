git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    tag = "0.10.0",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains(go_version="1.9.4")

# Get the Google API annotations, needed for gRPC-gateway.
# See https://stackoverflow.com/questions/47930973 for a bit of explanation.
new_http_archive(
    name = "googleapi",
    url = "https://github.com/googleapis/googleapis/archive/79f9041f4ad4984bf57d9ac5f9cec2ac506c3b48.zip",
    strip_prefix = "googleapis-79f9041f4ad4984bf57d9ac5f9cec2ac506c3b48/",
    build_file="BUILD.googleapi"
)

load("@io_bazel_rules_go//proto:def.bzl", "proto_register_toolchains")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_grpc_library")
