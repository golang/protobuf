load("@io_bazel_rules_go//go:def.bzl", "go_library")

# TODO(golang-protobuf-team): use genrule to generate from .proto
# Get protoc and make test work.
go_library(
    name = "go_default_library",
    testonly = 1,
    srcs = ["test.pb.go"],
    visibility = ["//proto:__subpackages__"],
    deps = ["//proto:go_default_library"],
)
