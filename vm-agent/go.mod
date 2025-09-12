module github.com/stackrox/stackrox/vm-agent

go 1.24.0

toolchain go1.24.4

require (
	github.com/mdlayher/vsock v1.2.1
	github.com/stackrox/rox v0.0.0
	google.golang.org/protobuf v1.36.8
)

replace github.com/stackrox/rox => ../

require (
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240409071808-615f978279ca // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/grpc v1.75.0 // indirect
)
