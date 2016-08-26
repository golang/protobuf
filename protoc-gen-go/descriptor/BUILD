load("@io_bazel_rules_go//go:def.bzl", "go_library")

# TODO(golang-protobuf-team): a genrule to generate from .proto
go_library(
    name = "go_default_library",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
    deps = ["//proto:go_default_library"],
)
