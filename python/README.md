
### Trickle Python Client and Samples

Publish and subscribe clients are are in `/trickle`

## Sample Programs

### main.sh
Subscribes to a video channel, converts the video into images, re-encodes the images again

Shell script assumes the publish url is subscribe url with an `-out` prefix

Requires ffmpeg
```
./main.sh <subscribe-url>
```

### image.sh

Publish an image at 30fps and subscribes back to the published stream

Assumes the publish channel is `http://localhost:2939/image-test`
```
./image.sh <path-to-image>
```

### read2pipe
Subscribes to a stream and writes the result to stdout.
```
python read2pipe.py --url=<url> > foo.ts
```

### echo
Subscribes to a channel and re-publishes it as-is.

The shell script assumes the publish URL is the subscribe URL with a `-out` prefix.
```
./echo.sh <subscribe-url>
```

### text-publisher
Publishes a text file, with a new segment every 20 lines.
```
python text-publisher.py --stream <channel name> --local <path to text file>
```

### segment-writer
Subscribes to a channel and writes each segment to a file.

The file path is hardcoded to `read-%d.ts` at the moment.
```
python segment-writer.py --stream=<subscribe-url>
```

### stress testing
Python publisher, golang subscriber.
Includes a log analysis tool to make sure both clients agree.

Python:
```
cd python && ./python3 stress-publisher.py --stream=<stream-name> --count=25 --segments=200 2>&1 | tee ../out-publisher.log
```

Golang:
```
go run cmd/stress-subscriber/stress_subscriber.go --stream=<stream-name> --count 25 2>&1 | tee out-subscriber.log
```

Logs:
```
go run cmd/stress-subscriber/compare_logs.go out-publisher.log out-subscriber.log
```
