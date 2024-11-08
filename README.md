# Trickle Protocol

Trickle is a segmented publish-subscribe protocol that streams data in realtime, mainly over HTTP.

Breaking this down:

1. Data streams are called *channels* , Channels are content agnostic - the data can be video, audio, JSON, logs, a custom format, etc.

2. Segmented: Data for each channel is sent as discrete *segments*. For example, a video may use a segment for each GOP, or an API may use a segment for each JSON message, or a logging application may split segments on time intervals. The boundaries of each segment are application defined, but segments are preferably standalone such that a request for a single segment should return usable content on its own.

3. Publish-subscribe: Publishers push segments, and subscribers pull them.

4. Streaming: Content can be sent by publishers and received by subscribers in real-time as it is generated - data is *trickled* out.

5. HTTP: Trickle is tied to HTTP with GET / POST semantics, status codes, path based routing, metadata in headers, etc. However, other implementations are possible, such as a local, in-memory trickle client.

### Protocol Specification

TODO: more details

Publishers POST to /channel-name/seq

Subscribers GET to /channel-name/seq

The `channel-name` is any valid HTTP path part.

The `seq` is an integer sequence number indicating the segment, and must increase sequentially without gaps.

As data is published to `channel-name/seq`,the server will relay the data to all subscribers for `channel-name/seq`.

Clients are responsible for maintaining their own position in the sequence.

Servers may opt to keep the last N segments for subscribers to catch up on.

Servers will 404 if a `channel-name` or a `seq` does not exist.

Clients may pre-connect the next segment in order to set up the resource and minimize connection set-up time.

Publishers should only actively send data to one `seq` at a time, although they may still pre-connect to `seq + 1`

Publishers do not have to push content immeditely after preconnecting, however the server should have some reasonable timeout to avoid excessive idle connections.

If a subscriber retrieves a segment mid-publish, the server should return all the content it has up until that point, and trickle down the rest as it receives it.

If a timeout has been hit without sending (or receiving) content, the publisher (or subscriber) can re-connect to the same `seq`. (TODO; also indicate timeout via signaling)

Servers may offer some grace with leading sequence numbers to avoid data races, eg allowing a GET for `seq+1` if a publisher hasn't yet preconnected that number.

Publishers are responsible for segmenting content (if necessary) and subscribers are responsible for re-assembling content (if necessary)

Subscribers can initiate a subscribe with a `seq` of -1 to retrieve the most recent publish. With preconnects, the subscriber may be waiting for the *next* publish. For video this allows clients to eg, start streaming at the live edge of the next GOP.

Subscribers can retrieve the current `seq` with the `Lp-Trickle-Seq` metadata (HTTP header). This is useful in case `-1` was used to initiate the subscription; the subscribing client can then pre-connect to `Lp-Trickle-Seq + 1`

Subscribers can initiate a subscribe with a `seq` of -N to get the Nth-from-last segment. (TODO)

The server should send subscribers `Lp-Trickle-Size` metadata to indicate the size of the content up until now. This allows clients to know where the live edge is, eg video implementations can decode-and-discard frames up until the edge to achieve immediate playback without waiting for the next segment. (TODO)

The server currently has a special changefeed channel named `_changes` which will send subscribers updates on streams that are added and removed. The changefeed is disabled by default.

## Sample Programs

The base trickle tools require golang 1.22+

### Trickle Server

This spins up a new trickle server on http://localhost:2939. Changefeeds are enabled on this server.

```
make trickle-server
```

#### Options
* `path`: Base path for the trickle server. Eg, `path=foo` makes the trickle server respond to `http://localhost:2939/foo`

### Playback Trickle Video Streams

Requires ffplay
```
make play stream=<stream-name>
```

#### Options
* `url`: URL of the trickle server if not localhost


### Send trickled data to standard output

```
go run cmd/read2pipe/*.go --stream <stream-name>
```

#### Options
* `--url` : URL of the trickle server if not localhost

### Trickle Video File Publisher

Requires ffmpeg

```
make publisher-ffmpeg in=<in-file> stream=<trickle-stream-name>
```

#### Options
* `url`: URL of the trickle server if not localhost

### Trickle Live Video Publisher

Waits for an incoming video stream from MediaMTX and publishes it as a trickle stream under the same name.

Requires ffmpeg and MediaMTX

```
make publisher
```

#### Options
* `url`: URL of the trickle server if not localhost

### Trickle Live Video Publisher and Subscriber


Waits for an incoming video stream from MediaMTX and publishes it as a trickle stream under the same name. Also listens for new trickle streams via changefeed and sends any trickle streams that end with `-out` into MediaMTX with the same name.

Requires ffmpeg, and MediaMTX and a trickle server with changefeeds enabled.

```
make pubsub-out
```

#### Options
* `url`: URL of the trickle server if not localhost

### Write segments to a file

```
make subscriber-example stream=<stream-name>
```


### MediaMTX

To listen for incoming media streams, the trickle sample apps expect [MediaMTX](https://github.com/bluenviron/mediamtx) to be running alongside the apps.

#### Configuring MediaMTX

MediaMTX is a media server that is easy to set up and run. Download a pre-built [release](https://github.com/bluenviron/mediamtx/releases/tag/v1.9.3). Use the `mediamtx.yml` configuration file that is included in the http-trickle repo [here](https://github.com/j0sh/http-trickle/blob/main/mediamtx.yml).

For completeness, here are the major differences in the `mediamtx.yml` configuration from the base:

```diff
diff --git b/mediamtx.yml a/mediamtx.yml
index c3aed76..cf7c60c 100644
--- b/mediamtx.yml
+++ a/mediamtx.yml
@@ -376,8 +376,8 @@ webrtcAdditionalHosts: []
 # ICE servers. Needed only when local listeners can't be reached by clients.
 # STUN servers allows to obtain and share the public IP of the server.
 # TURN/TURNS servers forces all traffic through them.
-webrtcICEServers2: []
-  # - url: stun:stun.l.google.com:19302
+webrtcICEServers2:
+  - url: stun:stun.l.google.com:19302
   # if user is "AUTH_SECRET", then authentication is secret based.
   # the secret must be inserted into the password field.
   # username: ''
@@ -643,7 +643,7 @@ pathDefaults:
   #   a regular expression.
   # * MTX_SOURCE_TYPE: source type
   # * MTX_SOURCE_ID: source ID
-  runOnReady:
+  runOnReady: curl -L -X POST http://localhost:2938/$MTX_PATH
   # Restart the command if it exits.
   runOnReadyRestart: no
   # Command to run when the stream is not available anymore.
```

#### Go Live with MediaMTX

Go live in the browser at `http://localhost:8889/<stream-name>/publish`

Go live via RTMP at `rtmp://localhost:8889/<stream-name>`

Trickle apps such as `publish` will automatically re-stream via trickle under the `<stream-name>` channel.

Refer to the MediaMTX documentation for more details.

Because trickle sample apps use ffmpeg RTMP to pull down streams from MediaMTX, the codecs used must match those what ffmpeg RTMP is capable of. Currently, this is limited to H.264 for video and G.722 for audio.
