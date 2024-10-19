module trickle

go 1.22.1

require github.com/livepeer/lpms v0.0.0-20240909171057-fe5aff1fa6a2

replace github.com/livepeer/lpms => ../../lpms

require golang.org/x/sys v0.26.0

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/livepeer/m3u8 v0.11.1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
