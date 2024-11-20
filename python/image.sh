set -eux

python image.py --image-path=$1 --subscribe-url "http://localhost:2939/image-test" --publish-url "http://localhost:2939/image-test"
