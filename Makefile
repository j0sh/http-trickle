
# Run `make play stream=streamname`
play:
	go run cmd/read2pipe/*.go --stream $(stream) | ffplay -probesize 32 -fflags nobuffer -flags low_delay -

trickle-server:
	go run cmd/trickle-server/*.go

# Listens for a connection from MediaMTX
# Run `make subscriber-example stream=streamname`
subscriber-example:
	go run cmd/subscriber-example.go trickle_subscriber.go trickle_publisher.go --stream $(stream)

OS := $(shell uname)

# Set the file name depending on the OS
ifeq ($(OS), Darwin)
	SELECT_FILE := select_darwin.go
else ifeq ($(OS), Linux)
	SELECT_FILE := select_linux.go
else
	$(error Unsupported OS: $(OS))
endif

pubsub:
	go run cmd/pubsub-mediamtx/*.go --out $(out)

publisher:
	go run cmd/publisher-mediamtx/main.go

tester:
	go run trickle_tester.go file2segment.go $(SELECT_FILE) trickle_publisher.go --stream $(stream) --local $(local)
