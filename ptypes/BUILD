load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

TESTS = glob(["*_test.go"])

go_library(
    name = "go_default_library",
    srcs = glob(
        ["*.go"],
        exclude = TESTS,
    ),
    visibility = ["//visibility:public"],
    deps = [
        "//proto:go_default_library",
        "//ptypes/any:go_default_library",
        "//ptypes/duration:go_default_library",
        "//ptypes/timestamp:go_default_library",
    ],
)

go_test(
    name = "ptypes_test",
    srcs = TESTS,
    library = ":go_default_library",
    deps = ["//protoc-gen-go/descriptor:go_default_library"],
)
