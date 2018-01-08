load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
    ],
    importpath = "github.com/livegrep/livegrep/cmd/livegrep-update-blame-cache",
    visibility = ["//visibility:public"],
    deps = [
        "//blameworthy:go_default_library",
        "@com_github_boltdb_bolt//:go_default_library",
    ],
)

go_binary(
    name = "strip-git-log",
    embed = [":go_default_library"],
    importpath = "github.com/livegrep/livegrep/cmd/livegrep-update-blame-cache",
    visibility = ["//visibility:public"],
)
