#!/bin/sh
set -eu

minio_host="${MINIO_DOCKER_HOST:-avatars.minio}"
minio_user="${MINIO_ROOT_USER:-minioadmin}"
minio_password="${MINIO_ROOT_PASSWORD:-minioadmin}"
bucket="${S3_BUCKET:-avatars}"

until mc alias set local "http://${minio_host}:9000" "${minio_user}" "${minio_password}"; do
  echo "waiting for minio at ${minio_host}:9000..."
  sleep 1
done

mc mb --ignore-existing "local/${bucket}"
echo "bucket ready: ${bucket}"
