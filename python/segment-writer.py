import asyncio
import aiofiles
import logging
import argparse
import sys

import trickle

# subscribes to a channel and writes each segment to a file
# NB: output file name right now is hard coded

async def write_chunks_to_file(input_stream, index, chunk_size=1024):
    """
    Asynchronously reads data in chunks from an input stream and writes it to a file with an incremented index.

    Args:
        input_stream (io.BufferedReader): The input stream to read data from.
        index (int): The index for the output file naming.
        chunk_size (int): The size of each chunk to read and write, in bytes. Default is 1024.
    """
    if None == index:
        logging.error("trickle sequence was none")
        sys.exit(1)


    # Create the filename using the index
    output_file_path = f"read-{index}.ts"

    # Open the output file asynchronously and write chunks to it
    async with aiofiles.open(output_file_path, 'wb') as output_file:
        while True:
            # Read a chunk from the input stream
            chunk = await input_stream.read(chunk_size)

            # If chunk is empty, end of stream is reached
            if not chunk:
                break

            # Write the chunk to the output file asynchronously
            await output_file.write(chunk)

    logging.info(f"Wrote to file {output_file_path}")


async def main(subscribe_url):
    subscriber = trickle.trickle_subscriber.TrickleSubscriber(url=subscribe_url)
    while True:
        segment = None
        try:
            segment = await subscriber.next()
            if not segment:
                break # complete
            if segment.eos():
                break # complete
            await write_chunks_to_file(segment, segment.seq(), chunk_size = 32 * 1024)
        except Exception as e:
            logging.error(f"Error writing to file: {e}")
            sys.exit(1)

if __name__ == "__main__":

    logging.basicConfig(
        format='%(asctime)s %(levelname)-8s %(message)s',
        level=logging.INFO,
        datefmt='%Y-%m-%d %H:%M:%S')

    parser = argparse.ArgumentParser(description="Write each segment to a file named 'read-%d.ts'")
    parser.add_argument(
        "--stream", type=str, required=True, help="stream name"
    )
    args = parser.parse_args()

    try:
        asyncio.run(
            main("http://localhost:2939/"+args.stream)
        )
    except Exception as e:
        logging.error(f"Fatal error in main: {e}")
        logging.error(f"Traceback:\n{''.join(traceback.format_tb(e.__traceback__))}")
        sys.exit(1)
