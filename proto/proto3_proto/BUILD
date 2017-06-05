load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    testonly = 1,
    srcs = glob(["*.go"]),
    visibility = ["//visibility:public"],
    deps = [
        "//proto:go_default_library",
        "//proto/testdata:go_default_library",
        "//ptypes/any:go_default_library",
    ],
)
