#!/bin/sh
set -e

# Populates a specified Docker registry node with Docker images.
# Arguments:
# 1. Docker registry node to SSH to.
# 2. Image to base the new image on (they will share no layers). If the image is
#    supposed to be created from scratch, "scratch" can be used here.
# 3. Prefix of the new image's name.
# The rest of the arguments are pairs which specify the amount of images to
# create and the size of these images in MiB.
# Example: registry node is the first node in the cluster, prefix is "img", and
#          the rest of the arguments are 5 10 3 20. This creates five images
#          named 10.0.1.1/img-10-x (where x is [0, 4]) of size 10MiB, and three
#          images named 10.0.1.1/img-20-y (where y is [0, 2]) of size 20MiB.
# Note: if the script fails with no error messages, there might be some in
#       /tmp/dirigent-build/log.txt on the registry node.

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

source "$DIR/common.sh"
source "$DIR/setup.cfg"

# Arguments
REMOTE_HOST="$1"
BASE_IMAGE="$2"
NEW_IMAGE_PREFIX="$3"
shift 3

# Derived variables
# TODO: Make this smarter
REGISTRY_ADDR=$(RemoteExec "$REMOTE_HOST" "ip addr show | grep 'inet 10[.]0[.]' | tr -s ' ' | cut -d' ' -f3 | cut -d/ -f1")

RemoteExec "$REMOTE_HOST" "sudo apt-get install docker-buildx" > /dev/null

base_image_size_bytes=0
if [ "$BASE_IMAGE" != "scratch" ]
then
    RemoteExec "$REMOTE_HOST" "docker pull '$BASE_IMAGE'" > /dev/null
    base_image_size_bytes=$(RemoteExec "$REMOTE_HOST" "docker save '$BASE_IMAGE' | gzip | wc -c")
fi

while [ "$#" -gt 1 ]
do
    num_images="$1"
    image_size_mib="$2"
    shift 2

    image_size_bytes=$(( $image_size_mib * 1024 * 1024 ))
    if [ $base_image_size_bytes -gt $image_size_bytes ]
    then
        base_image_size_mib=$(( $base_image_size_bytes / 1024 / 1024 ))
        echo "Base image is $base_image_size_bytes B ($base_image_size_mib MiB) gzipped."
        echo "Cannot create image smaller than the base image!"
        exit 2
    fi
    missing_size=$(( $image_size_bytes - $base_image_size_bytes ))
    missing_count=$(( $missing_size / 64 / 1024 ))

    RemoteExec "$REMOTE_HOST" "
    set -e
    mkdir -p /tmp/dirigent-build
    for (( i = 0; i < $num_images; i++ ))
    do
        repository_name=\"$NEW_IMAGE_PREFIX-$image_size_mib-\$i\"
        if curl -f -X GET \"http://$REGISTRY_ADDR/v2/\$repository_name/tags/list\" > /dev/null 2>&1
        then
            echo \"Image \$repository_name already in registry, skipping.\"
            continue
        fi
        image_name=\"$REGISTRY_ADDR/\$repository_name:latest\"
        echo \"Creating image \$image_name\"
        if [ \"$BASE_IMAGE\" == \"scratch\" ]
        then
            cat > /tmp/dirigent-build/Dockerfile <<EOF
            FROM alpine AS builder
            RUN dd if=/dev/urandom of=/dirigent-create-image-\$i bs=64k \"count=$missing_count\"

            FROM scratch
            COPY --from=builder /dirigent-create-image-\$i /dirigent-create-image-\$i
EOF
        else
            cat > /tmp/dirigent-build/Dockerfile <<EOF
            FROM $BASE_IMAGE AS builder
            RUN dd if=/dev/urandom of=/dirigent-create-image-\$i bs=64k \"count=$missing_count\"

            FROM scratch
            COPY --from=builder / /
            CMD /server
EOF
        fi
        docker build -t \"\$image_name\" /tmp/dirigent-build > /tmp/dirigent-build/log.txt 2>&1
        docker push \"\$image_name\" > /tmp/dirigent-build/log.txt 2>&1
        docker image prune -f > /tmp/dirigent-build/log.txt 2>&1
    done
    rm -rf /tmp/dirigent-build
    "
done

echo "Success!"
