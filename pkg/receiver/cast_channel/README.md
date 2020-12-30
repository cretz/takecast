Proto file from:

https://chromium.googlesource.com/openscreen/+/refs/heads/master/cast/common/channel/proto/cast_channel.proto

To generate, with protoc and protoc-gen-go on the path, run:

    protoc --go_out=. cast_channel.proto