load("@io_bazel_rules_go//go:def.bzl", "go_library")

filegroup(
    name = "go_default_library_protos",
    srcs = glob(["*.proto"]),
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
    deps = ["//proto:go_default_library"],
)
