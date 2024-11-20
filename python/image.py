import argparse
import asyncio
import logging
import sys
import traceback
import pathlib

import trickle

# takes an image, publishes it at 30fps, and subscribes to some result
# try subscribing to your own publish channel!

async def main(image_path:str, subscribe_url:str, publish_url:str):

    image=pathlib.Path(image_path).read_bytes()

    # take actual mimetype here if this differs
    publisher = trickle.trickle_publisher.TricklePublisher(url=publish_url, mime_type="image/jpeg")
    await publisher.create()

    subscriber = trickle.trickle_subscriber.TrickleSubscriber(url=subscribe_url)

    async def run_publish():
        fps=30
        for i in range(0, fps):
            try:
                async with await publisher.next() as segment:
                    await segment.write(image)
            except Exception as e:
                logging.error(f"Error running loop: {e}")
                break
            await asyncio.sleep(1000/fps/1000)
        await publisher.close()

    async def run_subscribe():
        while True:
            segment = await subscriber.next()
            if not segment:
                # publish has completed
                return
            logging.info(f'Got segment {segment.seq()}')
            while segment:
                chunk = await segment.read()
                if not chunk:
                    # image is complete, do something with it
                    break

    t1=asyncio.create_task(run_publish())
    t2=asyncio.create_task(run_subscribe())
    await asyncio.gather(t1, t2)

if __name__ == "__main__":

    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.INFO,
        datefmt='%Y-%m-%d %H:%M:%S')

    parser = argparse.ArgumentParser(description="trickle echo")
    parser.add_argument(
        "--image-path", type=str, required=True, help="image path"
    )
    parser.add_argument(
        "--subscribe-url", type=str, required=True, help="url to pull incoming streams"
    )
    parser.add_argument(
        "--publish-url", type=str, required=True, help="url to push outgoing streams"
    )
    args = parser.parse_args()

    try:
        asyncio.run(
            main(args.image_path, args.subscribe_url, args.publish_url)
        )
    except Exception as e:
        logging.error(f"Fatal error in main: {e}")
        logging.error(f"Traceback:\n{''.join(traceback.format_tb(e.__traceback__))}")
        sys.exit(1)
