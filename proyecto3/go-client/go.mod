module go-client

go 1.26.8

replace wartweets-proto => ../proto

require (
	google.golang.org/grpc v1.84.0
	wartweets-proto v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
