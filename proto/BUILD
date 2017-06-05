load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

LIBRARY_TESTS = [
    "size2_test.go",
    "message_set_test.go",
]

TESTS = glob(
    ["*_test.go"],
    exclude = LIBRARY_TESTS,
)

REFLECT = ["pointer_reflect.go"]

UNSAFE = ["pointer_unsafe.go"]

config_setting(
    name = "golang_appengine",
    values = {"define": "golang_appengine=1"},
)

go_library(
    name = "go_default_library",
    srcs = glob(
        ["*.go"],
        exclude = TESTS + REFLECT + UNSAFE + LIBRARY_TESTS,
    ) + select({
        ":golang_appengine": REFLECT,
        "//conditions:default": UNSAFE,
    }),
    visibility = ["//visibility:public"],
)

go_test(
    name = "proto",
    srcs = LIBRARY_TESTS,
    library = ":go_default_library",
)

go_test(
    name = "proto_test",
    srcs = TESTS,
    deps = [
        ":go_default_library",
        "//proto/proto3_proto:go_default_library",
        "//proto/testdata:go_default_library",
        "//ptypes/any:go_default_library",
    ],
)
