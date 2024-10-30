import argparse
import asyncio
import json
import logging
import signal
import sys
import os
import traceback
import queue
from typing import List

import media

async def main(subscribe_url: str, publish_url: str, params: dict):
    logger = logging.getLogger(__name__)

    image_queue = queue.Queue()
    def image_callback(image):
        if not image:
            logging.info("Image callback received None")
            image_queue.put(image)
            return
        # TODO send image to inference, maybe via asyncio.Queue
        # For now we just loop it back to the output
        try:
            image_queue.put(image, block=False)
            #logging.info(f"Image queue put in {len(image)}")
        except Full:
            logging.info("Image queue full; dropping image")
        except ShutDown:
            logging.info("Shut down!")

    # Set up async signal handling for SIGINT and SIGTERM
    stop_event = asyncio.Event()
    def stop_signal_handler():
        logging.info("Stopping program due to received signal")
        stop_event.set()
    loop = asyncio.get_running_loop()
    loop.add_signal_handler(signal.SIGINT, stop_signal_handler)
    loop.add_signal_handler(signal.SIGTERM, stop_signal_handler)

    try:
        media_task = asyncio.create_task(media.preprocess(subscribe_url, image_callback))
        publish_task = asyncio.create_task(media.postprocess(publish_url, image_queue))
    except Exception as e:
        logging.error(f"Error starting socket handler or HTTP server: {e}")
        logging.error(f"Stack trace:\n{traceback.format_exc()}")
        raise e

    await block_until_signal([signal.SIGINT, signal.SIGTERM])
    await stop_event.wait()
    await media_task.cancel()

    try:
        await media_task
    except Exception as e:
        logging.error(f"Error stopping room handler: {e}")
        logging.error(f"Stack trace:\n{traceback.format_exc()}")
        raise e


async def block_until_signal(sigs: List[signal.Signals]):
    loop = asyncio.get_running_loop()
    future: asyncio.Future[signal.Signals] = loop.create_future()

    def signal_handler(sig, _):
        logging.info(f"Received signal: {sig}")
        loop.call_soon_threadsafe(future.set_result, sig)

    for sig in sigs:
        signal.signal(sig, signal_handler)
    return await future

if __name__ == "__main__":

    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.INFO,
        datefmt='%Y-%m-%d %H:%M:%S')

    logging.info("JOSH in main")
    parser = argparse.ArgumentParser(description="Infer process to run the AI pipeline")
    parser.add_argument(
        "--initial-params", type=str, default="{}", help="Initial parameters for the pipeline"
    )
    parser.add_argument(
        "--subscribe-url", type=str, required=True, help="url to pull incoming streams"
    )
    parser.add_argument(
        "--publish-url", type=str, required=True, help="url to push outgoing streams"
    )
    args = parser.parse_args()
    try:
        params = json.loads(args.initial_params)
    except Exception as e:
        logging.error(f"Error parsing --initial-params: {e}")
        sys.exit(1)

    try:
        asyncio.run(
            main(args.subscribe_url, args.publish_url, params)
        )
    except Exception as e:
        logging.error(f"Fatal error in main: {e}")
        logging.error(f"Traceback:\n{''.join(traceback.format_tb(e.__traceback__))}")
        sys.exit(1)

