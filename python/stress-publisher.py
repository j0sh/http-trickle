import argparse
import asyncio
import datetime
import hashlib
import random
import sys

from trickle import TricklePublisher

BYTES_PER_SEGMENT   = 588_576               # 500kb
SEGMENT_DURATION_S  = 2.0                     # seconds
CHUNK_SIZE          = 16 * 1024               # 16 KiB → 64 chunks / segment
CHUNKS_PER_SEGMENT  = BYTES_PER_SEGMENT // CHUNK_SIZE
SLEEP_PER_CHUNK     = SEGMENT_DURATION_S / CHUNKS_PER_SEGMENT
JITTER_MAX_S        = 2.0          # spread start-ups over ≤2 s


async def run_publisher(idx: int, base_url: str, stream_name: str, segments: int) -> None:
    """
    Continuously publishes SEGMENTS_PER_PUB segments of 1 MiB each,
    spread over SEGMENT_DURATION_S seconds, and prints a SHA-256
    of the payload for every segment to stderr.
    """
    # Give each publisher its own unique stream path
    writer = TricklePublisher(f"{base_url}/{stream_name}_{idx}", "application/octet-stream")

    # stagger the publishers
    await asyncio.sleep(random.uniform(0, JITTER_MAX_S))

    try:
        for seg_no in range(segments):
            async with await writer.next() as segment:
                hasher = hashlib.sha256()
                seq = segment.seq()

                for _ in range(CHUNKS_PER_SEGMENT):
                    chunk = random.randbytes(CHUNK_SIZE)       # non-crypto randomness is fine here
                    hasher.update(chunk)
                    await segment.write(chunk)
                    await asyncio.sleep(SLEEP_PER_CHUNK)

            # emit hash for the finished segment
            print(
                f"{datetime.datetime.now().strftime('%Y/%m/%d %H:%M:%S.%f')[:-3]} "
                f"{idx:02d}-{seq:04d} "
                f"SHA-256={hasher.hexdigest()}",
                file=sys.stderr,
                flush=True,
            )
    finally:
        await writer.close()


async def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default="http://localhost:2939",  help="Base URL of the Trickle server")
    parser.add_argument("--stream", required=True, help="Base stream name (publisher index is appended)")
    parser.add_argument("--count", default=25, help="Number of concurrent publishers")
    parser.add_argument("--segments", default=2000, help="Number of segments per publishers")
    args = parser.parse_args()

    # Launch publishers concurrently
    publishers = [
        asyncio.create_task(run_publisher(i, args.url, args.stream, int(args.segments)))
        for i in range(int(args.count))
    ]
    res = await asyncio.gather(*publishers, return_exceptions=True)
    print(f"Gather result: {res}")


if __name__ == "__main__":
    asyncio.run(main())
