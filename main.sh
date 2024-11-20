set -eux

python main.py --subscribe-url "$1" --publish-url "$1-out"
