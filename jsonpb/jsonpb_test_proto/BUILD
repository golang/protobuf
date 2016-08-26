load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
    deps = [
        "//proto:go_default_library",
        "//ptypes/any:go_default_library",
        "//ptypes/duration:go_default_library",
        "//ptypes/struct:go_default_library",
        "//ptypes/timestamp:go_default_library",
        "//ptypes/wrappers:go_default_library",
    ],
)
