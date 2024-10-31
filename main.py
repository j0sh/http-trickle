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

import trickle

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
    loop = asyncio.get_running_loop()
    def stop_signal_handler():
        logging.info("Stopping program due to received signal")
        loop.stop()
    loop.add_signal_handler(signal.SIGINT, stop_signal_handler)
    loop.add_signal_handler(signal.SIGTERM, stop_signal_handler)

    try:
        subscribe_task = asyncio.create_task(trickle.run_subscribe(subscribe_url, image_callback))
        publish_task = asyncio.create_task(trickle.run_publish(publish_url, image_queue))
    except Exception as e:
        logging.error(f"Error starting socket handler or HTTP server: {e}")
        logging.error(f"Stack trace:\n{traceback.format_exc()}")
        raise e

    try:
        await asyncio.gather(subscribe_task, publish_task)
    except Exception as e:
        logging.error(f"Error stopping room handler: {e}")
        logging.error(f"Stack trace:\n{traceback.format_exc()}")
        raise e


if __name__ == "__main__":

    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.INFO,
        datefmt='%Y-%m-%d %H:%M:%S')

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
