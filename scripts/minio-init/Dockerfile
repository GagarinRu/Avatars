FROM quay.io/minio/mc:latest AS mcimg

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=mcimg /usr/bin/mc /usr/bin/mc
COPY init-bucket.sh /init-bucket.sh
RUN sed -i 's/\r$//' /init-bucket.sh && chmod +x /init-bucket.sh

ENTRYPOINT ["/bin/sh", "/init-bucket.sh"]
