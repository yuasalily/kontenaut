#!/bin/sh
set -eu

PREFIX="wrapper-test"

echo "=== GLOBAL CLEANUP (ALL containers & images) ==="

# すべてのコンテナ削除
docker ps -aq | xargs -r docker rm -f

# すべてのイメージ削除
docker images -aq | xargs -r docker rmi -f

echo "=== Create images (no Dockerfile) ==="

# ---- deletable images (4) ----
i=1
while [ $i -le 4 ]; do
  docker run -d \
    --name "${PREFIX}-tmp-img-deletable-$i" \
    alpine sh -c "echo image-deletable-$i > /file && sleep 2"

  docker commit \
    "${PREFIX}-tmp-img-deletable-$i" \
    "${PREFIX}-image-deletable-$i"

  docker rm -f "${PREFIX}-tmp-img-deletable-$i"
  i=$((i + 1))
done

# ---- non-deletable image (in use) ----
docker run -d \
  --name "${PREFIX}-tmp-img-nondeletable" \
  alpine sh -c "echo image-nondeletable > /file && sleep 2"

docker commit \
  "${PREFIX}-tmp-img-nondeletable" \
  "${PREFIX}-image-nondeletable"

docker rm -f "${PREFIX}-tmp-img-nondeletable"

echo "=== Create containers (deletable / non-deletable) ==="

# ---- deletable containers (stopped, 5) ----
i=1
while [ $i -le 5 ]; do
  docker create \
    --name "${PREFIX}-container-deletable-$i" \
    "${PREFIX}-image-deletable-1"
  i=$((i + 1))
done

# ---- non-deletable containers (running, 2) ----
docker run -d \
  --name "${PREFIX}-container-nondeletable-1" \
  "${PREFIX}-image-deletable-1" sh -c "sleep 600"

docker run -d \
  --name "${PREFIX}-container-nondeletable-2" \
  "${PREFIX}-image-deletable-1" sh -c "sleep 600"

# ---- container that blocks image deletion ----
docker create \
  --name "${PREFIX}-container-uses-image-nondeletable" \
  "${PREFIX}-image-nondeletable"

echo "=== Create log-retaining containers ==="

# ---- stopped but logs remain ----
docker run \
  --name "${PREFIX}-container-with-logs-stopped" \
  alpine sh -c "
    echo '[INFO] start';
    echo '[DEBUG] doing something';
    echo '[INFO] finished';
  "

# ---- running with continuous logs ----
docker run -d \
  --name "${PREFIX}-container-with-logs-running" \
  alpine sh -c "
    i=1;
    while true; do
      echo \"[LOG] tick looooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong \$i\";
      i=\$((i+1));
      sleep 0.005;
    done
  "

echo
echo "=== Delete & Log Test Summary ==="
echo
echo "Containers (deletable, batch OK):"
echo "  ${PREFIX}-container-deletable-1..5"
echo
echo "Containers (non-deletable without -f):"
echo "  ${PREFIX}-container-nondeletable-1"
echo "  ${PREFIX}-container-nondeletable-2"
echo
echo "Containers (log test):"
echo "  ${PREFIX}-container-with-logs-stopped  (docker logs OK)"
echo "  ${PREFIX}-container-with-logs-running  (docker logs -f OK)"
echo
echo "Images (deletable, batch OK):"
echo "  ${PREFIX}-image-deletable-1..4"
echo
echo "Image (non-deletable):"
echo "  ${PREFIX}-image-nondeletable"
echo
echo "=== Current state ==="
docker ps -a | grep "${PREFIX}"
docker images | grep "${PREFIX}"
