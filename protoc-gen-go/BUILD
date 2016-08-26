load("@io_bazel_rules_go//go:def.bzl", "go_binary")

go_binary(
    name = "protoc-gen-go",
    srcs = glob(
        ["*.go"],
        exclude = ["doc.go"],
    ),
    visibility = ["//visibility:public"],
    deps = [
        "//proto:go_default_library",
        "//protoc-gen-go/generator:go_default_library",
        "//protoc-gen-go/grpc:go_default_library",
    ],
)
