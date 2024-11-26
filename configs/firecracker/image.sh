#!/bin/bash
#
# MIT License
#
# Copyright (c) 2024 EASL
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

set -x
# get args
SOURCE_PATH=""
DEST_PATH=""
APP_NAME=""
ROOTFS=""
while getopts "a:d:s:r:" optname; do
    case "${optname}" in
        d)
            DEST_PATH=${OPTARG}
            ;;
        a)
            APP_NAME=${OPTARG}
            ;;
        s)
            SOURCE_PATH=${OPTARG}
            ;;
        r)
            ROOTFS=${OPTARG}
            ;;
        *)
            echo "Unknown option ${optname}"
            exit 1
    esac
done
if [ "$APP_NAME" = "" ]; then
    APP_NAME="app"
fi
if [ "$DEST_PATH" = "" ]; then
    DEST_PATH=$(pwd)
fi
if [ "$ROOTFS" = "" ]; then
    ROOTFS="$(pwd)/rootfs.ext4"
fi
if [ "$SOURCE_PATH" = "" ]; then
    echo "Need source path"
    exit 1
fi

build_go_function(){
    export PATH=$PATH:/usr/local/go/bin
    go build -buildvcs=false -tags netgo -ldflags '-extldflags "-static"' -o "$DEST_PATH/$APP_NAME" || exit
    echo "Built go function"
}
#  build the application
(
    # Execute in subshell to avoid changing shell directory
    cd "$SOURCE_PATH" || exit
    # Build function based on source path contents
    # by checking if there is a *.go file in the source path, "no such file" errors are piped to /dev/null
    if [[ -n $(find "$SOURCE_PATH/workload.go" 2>/dev/null) ]]; then
        build_go_function
    else
        echo "Failed to find the workload for image building."
        exit 1
    fi
)

TMP_ROOTFS="/tmp/${ROOTFS:1}"

dd if=/dev/zero of="$ROOTFS" bs=1M count=64
mkfs.ext4 "$ROOTFS"
mkdir -p "$TMP_ROOTFS"
mount "$ROOTFS" "$TMP_ROOTFS"

SCRIPT_PATH=$(realpath ${BASH_SOURCE})
SCRIPT_DIR=$(dirname "$SCRIPT_PATH")

docker run -i --rm \
    -v "$TMP_ROOTFS":/rootfs \
    -v "$DEST_PATH/$APP_NAME:/usr/local/bin/agent" \
    -v "$SCRIPT_DIR/openrc-service.sh:/etc/init.d/agent" \
    alpine sh <"$SCRIPT_DIR"/setup-alpine.sh

# script can change ownership of openrc-service.sh
chown "$SUDO_UID:$SUDO_GID" "$SCRIPT_DIR/openrc-service.sh"

umount "$TMP_ROOTFS"
rm -r "$TMP_ROOTFS"
chown "$SUDO_UID" "$ROOTFS"
