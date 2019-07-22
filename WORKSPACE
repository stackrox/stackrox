load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/0.19.1/rules_go-0.19.1.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/0.19.1/rules_go-0.19.1.tar.gz",
    ],
    sha256 = "8df59f11fb697743cbb3f26cfb8750395f30471e9eabde0d174c3aebc7a1cd39",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains(go_version="1.12")

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.18.1/bazel-gazelle-0.18.1.tar.gz"],
    sha256 = "be9296bfd64882e3c08e3283c58fcb461fa6dd3c171764fcc4cf322f60615a9b",
)
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
gazelle_dependencies()

# Get the Google API annotations, needed for gRPC-gateway.
# See https://stackoverflow.com/questions/47930973 for a bit of explanation.
http_archive(
    name = "googleapi",
    url = "https://github.com/googleapis/googleapis/archive/79f9041f4ad4984bf57d9ac5f9cec2ac506c3b48.zip",
    strip_prefix = "googleapis-79f9041f4ad4984bf57d9ac5f9cec2ac506c3b48/",
    build_file="BUILD.googleapi"
)

load("@io_bazel_rules_go//proto:def.bzl", "proto_register_toolchains")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_grpc_library")
