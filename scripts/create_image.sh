#!/bin/bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

# Creates a function image with a specific size when compressed.
# Arguments:
# 1. Image to base the new image on (they will share no layers). If the image is
#    supposed to be created from scratch, "scratch" can be used here.
# 2. Repository to push the new image to.
# 3. Image's size in MiB.
# 4. (Optional) Image's tag (default: latest).

if [ $# -lt 3 ]
then
	echo "Usage: ./create_image.sh <base_image> <repository> <image_size_in_mib> [image_tag]"
	exit 1
fi

# Input arguments
BASE_IMAGE="$1"
REPOSITORY="$2"
IMAGE_SIZE_MIB="$3"
IMAGE_TAG="${4:-latest}"

# Derived variables
IMAGE_NAME="$REPOSITORY:$IMAGE_TAG"
IMAGE_SIZE_BYTES=$(( $IMAGE_SIZE_MIB * 1024 * 1024 ))

# Pull the base image and check its compressed size.
base_image_size_bytes=0
base_image_size_mib=0
if [ "$BASE_IMAGE" != "scratch" ]
then
	docker pull "$BASE_IMAGE"
	base_image_size_bytes=`docker save "$BASE_IMAGE" | gzip | wc -c`
	base_image_size_mib=$(( $base_image_size_bytes / 1024 / 1024 ))
	builder_image="$BASE_IMAGE"
	echo "Base image is $base_image_size_bytes B ($base_image_size_mib MiB) gzipped."
fi
# Error if we can't create an image with the requested size.
if [ $base_image_size_bytes -gt $IMAGE_SIZE_BYTES ]
then
	echo "Cannot create image smaller than the base image!"
	exit 2
fi
# Build a new image with an additional file filling it to specified size.
MISSING_SIZE=$(( $IMAGE_SIZE_BYTES - $base_image_size_bytes ))
if [ "$BASE_IMAGE" == "scratch" ]
then
	cat > "$DIR/Dockerfile.sized" <<EOF
	FROM alpine AS builder
	RUN dd if=/dev/urandom "of=/dirigent-create-image" "bs=$MISSING_SIZE" count=1

	FROM scratch
	COPY --from=builder /dirigent-create-image /dirigent-create-image
EOF
else
	cat > "$DIR/Dockerfile.sized" <<EOF
	FROM $builder_image AS builder
	RUN dd if=/dev/urandom "of=/dirigent-create-image" "bs=$MISSING_SIZE" count=1

	FROM scratch
	COPY --from=builder / /
	CMD /server
EOF
fi
docker build -t "$IMAGE_NAME" -f "$DIR/Dockerfile.sized" "$DIR" > /dev/null
# Push the image to a Docker repository
docker push "$IMAGE_NAME"
# Clean up
rm "$DIR/Dockerfile.sized"
