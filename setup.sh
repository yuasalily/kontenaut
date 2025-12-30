#!/usr/bin/env bash
set -euo pipefail

PREFIX="wrapper-test"
IMAGES=(
  "${PREFIX}-image-1"
  "${PREFIX}-image-2"
)
CONTAINERS=(
  "${PREFIX}-container-1"
  "${PREFIX}-container-2"
  "${PREFIX}-container-3"
)

echo "=== Cleanup phase ==="

# コンテナ削除（存在しなくてもOK）
for c in "${CONTAINERS[@]}"; do
  if docker ps -a --format '{{.Names}}' | grep -qx "$c"; then
    docker rm -f "$c"
  fi
done

# イメージ削除（存在しなくてもOK）
for i in "${IMAGES[@]}"; do
  if docker images --format '{{.Repository}}' | grep -qx "$i"; then
    docker rmi -f "$i"
  fi
done

echo "=== Create base containers ==="

# 一時コンテナ作成（イメージ作成用）
docker run -d --name "${PREFIX}-tmp-1" alpine sh -c "echo image1 > /image.txt && sleep 5"
docker run -d --name "${PREFIX}-tmp-2" alpine sh -c "echo image2 > /image.txt && sleep 5"

echo "=== Commit images (no Dockerfile) ==="

docker commit "${PREFIX}-tmp-1" "${IMAGES[0]}"
docker commit "${PREFIX}-tmp-2" "${IMAGES[1]}"

# 一時コンテナ削除
docker rm -f "${PREFIX}-tmp-1" "${PREFIX}-tmp-2"

echo "=== Create containers from images ==="

# 停止状態のコンテナ
docker create --name "${CONTAINERS[0]}" "${IMAGES[0]}"

# 起動状態のコンテナ
docker run -d --name "${CONTAINERS[1]}" "${IMAGES[0]}" sh -c "sleep 300"
docker run -d --name "${CONTAINERS[2]}" "${IMAGES[1]}" sh -c "sleep 300"

echo "=== Result ==="
docker ps -a --filter "name=${PREFIX}"
docker images | grep "${PREFIX}"

echo "Done."
