import asyncio
import aiofiles
import logging
import argparse
import sys

import trickle

# subscribe to a stream and write it to stdout

async def main(subscribe_url):
    subscriber = trickle.trickle_subscriber.TrickleSubscriber(url=subscribe_url)
    while True:
        segment = None
        try:
            segment = await subscriber.next()
            if not segment:
                break # complete
            while segment:
                chunk = await segment.read()
                if not chunk:
                    break
                # Write the binary data to stdout (sys.stdout.buffer for binary output)
                sys.stdout.buffer.write(chunk)
                sys.stdout.buffer.flush()  # Ensure data is flushed to stdout
        except Exception as e:
            logging.error(f"Error writing to file: {e}")
            sys.exit(1)

if __name__ == "__main__":

    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.INFO,
        datefmt='%Y-%m-%d %H:%M:%S')

    parser = argparse.ArgumentParser(description="Infer process to run the AI pipeline")
    parser.add_argument(
        "--url", type=str, required=True, help="stream url"
    )
    args = parser.parse_args()

    try:
        asyncio.run(
            main(args.url)
        )
    except Exception as e:
        logging.error(f"Fatal error in main: {e}")
        logging.error(f"Traceback:\n{''.join(traceback.format_tb(e.__traceback__))}")
        sys.exit(1)
