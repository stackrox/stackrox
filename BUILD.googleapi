package(default_visibility=['//visibility:public'])

proto_library(
name = 'annotations_proto',
srcs = ['google/api/annotations.proto'],
deps = [
        ":http_proto",
        "@com_google_protobuf//:descriptor_proto"
    ],
)

proto_library(
    name = 'http_proto',
    srcs = ['google/api/http.proto']
)