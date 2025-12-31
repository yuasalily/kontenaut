#!/bin/sh
set -eu

PREFIX="wrapper-test"

echo "=== Cleanup phase ==="

# コンテナ全削除
docker ps -a --format '{{.Names}}' \
  | grep "^${PREFIX}" \
  | xargs -r docker rm -f

# イメージ全削除
docker images --format '{{.Repository}}' \
  | grep "^${PREFIX}" \
  | xargs -r docker rmi -f

echo "=== Create base images (no Dockerfile) ==="

# 削除できるイメージ用
i=1
while [ $i -le 4 ]; do
  docker run -d \
    --name "${PREFIX}-tmp-img-deletable-$i" \
    alpine sh -c "echo deletable-$i > /file && sleep 5"

  docker commit \
    "${PREFIX}-tmp-img-deletable-$i" \
    "${PREFIX}-image-deletable-$i"

  docker rm -f "${PREFIX}-tmp-img-deletable-$i"
  i=$((i + 1))
done

# 削除できないイメージ用
docker run -d \
  --name "${PREFIX}-tmp-img-nondeletable" \
  alpine sh -c "echo nondeletable > /file && sleep 5"

docker commit \
  "${PREFIX}-tmp-img-nondeletable" \
  "${PREFIX}-image-nondeletable"

docker rm -f "${PREFIX}-tmp-img-nondeletable"

echo "=== Create deletable containers (stopped) ==="

i=1
while [ $i -le 5 ]; do
  docker create \
    --name "${PREFIX}-container-deletable-$i" \
    "${PREFIX}-image-deletable-1"
  i=$((i + 1))
done

echo "=== Create non-deletable containers (running) ==="

docker run -d \
  --name "${PREFIX}-container-nondeletable-1" \
  "${PREFIX}-image-deletable-1" sh -c "sleep 600"

docker run -d \
  --name "${PREFIX}-container-nondeletable-2" \
  "${PREFIX}-image-deletable-1" sh -c "sleep 600"

echo "=== Create container that blocks image deletion ==="

docker create \
  --name "${PREFIX}-container-uses-image-nondeletable" \
  "${PREFIX}-image-nondeletable"

echo
echo "=== Delete test summary ==="
echo "Images (deletable, batch OK):"
echo "  ${PREFIX}-image-deletable-1"
echo "  ${PREFIX}-image-deletable-2"
echo "  ${PREFIX}-image-deletable-3"
echo "  ${PREFIX}-image-deletable-4"
echo
echo "Image (non-deletable):"
echo "  ${PREFIX}-image-nondeletable"
echo
echo "Containers (deletable):"
echo "  ${PREFIX}-container-deletable-1..5"
echo
echo "Containers (non-deletable without -f):"
echo "  ${PREFIX}-container-nondeletable-1"
echo "  ${PREFIX}-container-nondeletable-2"
echo
echo "=== Current state ==="
docker ps -a | grep "${PREFIX}"
docker images | grep "${PREFIX}"
