#!/bin/bash
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
    go build -tags netgo -ldflags '-extldflags "-static"' -o "$DEST_PATH/$APP_NAME" || exit
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

dd if=/dev/zero of="$ROOTFS" bs=1M count=128
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
