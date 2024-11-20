set -eux

python echo.py --subscribe-url "$1" --publish-url "$1-out"
