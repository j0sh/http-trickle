# Run `make reader stream=streamname`
reader:
	go run read2pipe.go trickle_reader.go trickle_writer.go --stream $(stream) | ffprobe -probesize 32 -fflags nobuffer -flags low_delay -

server:
	go run trickle_server.go

# Expects MediaMTX
writer:
	go run main.go rtmp2segment.go trickle_writer.go
