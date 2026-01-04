#!/bin/sh
set -eu

PREFIX="wrapper-test"
COMPOSE_DIR="./compose-test-env"
COMPOSE_FILE="docker-compose.test.yaml"
COMPOSE_PROJECT="test_kontenaut_compose"

echo "=== GLOBAL CLEANUP (ALL containers & images) ==="

docker ps -aq | xargs -r docker rm -f
docker images -aq | xargs -r docker rmi -f
docker network prune -f

echo "=== Create standalone images (no Dockerfile) ==="

i=1
while [ $i -le 3 ]; do
  docker run -d \
    --name "${PREFIX}-tmp-img-$i" \
    alpine sh -c "echo image-$i > /file && sleep 2"

  docker commit \
    "${PREFIX}-tmp-img-$i" \
    "${PREFIX}-image-deletable-$i"

  docker rm -f "${PREFIX}-tmp-img-$i"
  i=$((i + 1))
done

echo "=== Create standalone containers ==="

i=1
while [ $i -le 3 ]; do
  docker create \
    --name "${PREFIX}-container-deletable-$i" \
    "${PREFIX}-image-deletable-1"
  i=$((i + 1))
done

docker run -d \
  --name "${PREFIX}-container-nondeletable-1" \
  "${PREFIX}-image-deletable-1" sh -c "sleep 600"

echo "=== Create containers with logs ==="

docker run \
  --name "${PREFIX}-container-with-logs-stopped" \
  alpine sh -c "
    echo '[TEST][STANDALONE] start';
    echo '[TEST][STANDALONE] end';
  "

docker run -d \
  --name "${PREFIX}-container-with-logs-running" \
  alpine sh -c "
    i=1;
    while true; do
      echo \"[TEST][STANDALONE] tick \$i\";
      i=\$((i+1));
      sleep 2;
    done
  "

echo "=== Create TEST docker-compose environment ==="

rm -rf "${COMPOSE_DIR}"
mkdir -p "${COMPOSE_DIR}"

cat > "${COMPOSE_DIR}/${COMPOSE_FILE}" <<EOF
version: "3.9"

name: ${COMPOSE_PROJECT}

services:
  test_app:
    image: alpine
    labels:
      purpose: test
      suite: docker-sdk-wrapper
    command: >
      sh -c "while true; do
        echo '[TEST][COMPOSE][app] running';
        sleep 3;
      done"

  test_worker:
    image: alpine
    labels:
      purpose: test
      suite: docker-sdk-wrapper
    command: >
      sh -c "
        echo '[TEST][COMPOSE][worker] start';
        sleep 5;
        echo '[TEST][COMPOSE][worker] end';
      "
EOF

(
  cd "${COMPOSE_DIR}"
  docker compose -f "${COMPOSE_FILE}" up -d
)

echo
echo "=== TEST Environment Summary ==="
echo
echo "[Standalone]"
echo "  Containers : ${PREFIX}-container-*"
echo "  Images     : ${PREFIX}-image-*"
echo
echo "[Compose TEST]"
echo "  Project    : ${COMPOSE_PROJECT}"
echo "  File       : ${COMPOSE_DIR}/${COMPOSE_FILE}"
echo "  Containers : ${COMPOSE_PROJECT}-test_app-1"
echo "               ${COMPOSE_PROJECT}-test_worker-1"
echo "  Network    : ${COMPOSE_PROJECT}_default"
echo
echo "=== Current state ==="
docker ps -a
docker images
docker network ls
