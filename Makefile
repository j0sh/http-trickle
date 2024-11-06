
# Run `make play stream=streamname`
play:
	go run cmd/read2pipe/*.go $(if $(url),--url $(url)) --stream $(stream) | ffplay -probesize 32 -fflags nobuffer -flags low_delay -

trickle-server:
	go run cmd/trickle-server/*.go $(if $(path),--path $(path))

# Listens for a connection from MediaMTX
# Run `make subscriber-example stream=streamname`
subscriber-example:
	go run cmd/subscriber-example/*.go --stream $(stream)

publisher-ffmpeg:
	$(if $(in),, $(error in file is not set. Please provide in= as an argument))
	ffmpeg -loglevel warning -re -i $(in) -c copy -f mpegts - | go run cmd/publisher-ffmpeg/*.go --stream $(stream) $(if $(url),--url $(url))

pubsub-out:
	go run cmd/publisher-out/*.go $(if $(url),--url $(url))

pubsub-pipe:
	go run cmd/pubsub-mediamtx/*.go --out $(out)

publisher:
	go run cmd/publisher-mediamtx/main.go $(if $(url),--url $(url))
