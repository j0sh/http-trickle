import argparse
import asyncio
import logging
import signal
import sys
import os
import traceback
import queue

import trickle

# echo subscribes to a stream and re-publishes it

async def main(subscribe_url:str, publish_url:str):

    # Set up async signal handling for SIGINT and SIGTERM
    loop = asyncio.get_running_loop()
    def stop_signal_handler():
        logging.info("Stopping program due to received signal")
        loop.stop()
    loop.add_signal_handler(signal.SIGINT, stop_signal_handler)
    loop.add_signal_handler(signal.SIGTERM, stop_signal_handler)

    subscriber = trickle.trickle_subscriber.TrickleSubscriber(url=subscribe_url)
    publisher = trickle.trickle_publisher.TricklePublisher(url=publish_url, mime_type="video/mp2t")

    # publish a single segment, retrieving chunks from a queue
    async def run_publish(nextSegment, queue):
        async with await nextSegment() as segment:
            while True:
                chunk = await queue.get()
                if not chunk:
                    return
                await segment.write(chunk)

    # run subscriber, putting chunks for each segment into a queue
    while True:
        try:
            queue = asyncio.Queue()
            asyncio.create_task(run_publish(publisher.next, queue))
            segment = await subscriber.next()
            if not segment:
                await queue.put(None)
                break
            while segment:
                chunk = await segment.read(32 * 1024)
                await queue.put(chunk)
                if not chunk:
                    break
            await segment.close()
        except Exception as e:
            logging.error(f"Error running lloop: {e}")
            sys.exit(1)

if __name__ == "__main__":

    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.INFO,
        datefmt='%Y-%m-%d %H:%M:%S')

    parser = argparse.ArgumentParser(description="trickle echo")
    parser.add_argument(
        "--subscribe-url", type=str, required=True, help="url to pull incoming streams"
    )
    parser.add_argument(
        "--publish-url", type=str, required=True, help="url to push outgoing streams"
    )
    args = parser.parse_args()

    try:
        asyncio.run(
            main(args.subscribe_url, args.publish_url)
        )
    except Exception as e:
        logging.error(f"Fatal error in main: {e}")
        logging.error(f"Traceback:\n{''.join(traceback.format_tb(e.__traceback__))}")
        sys.exit(1)
