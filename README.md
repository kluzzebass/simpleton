# Simpleton
Simpleton HTTPd is a really simple web server that serves static content from a directory.

## Docker

```
docker run -p 80:80 -v /tmp/simpleton/content:/www kluzz/simpleton:latest
```

## Docker Compose

```
services:
  simpleton:
    image: kluzz/simpleton:latest
    container_name: "simpleton"
    ports:
      - "80:80"
    volumes:
      - ./www:/www:ro
      - ./logs:/var/log/simpleton
    environment:
      - SIMPLETON_LISTEN_ADDRS=:80
      - SIMPLETON_ACCESS_LOG_PATH=/var/log/simpleton/access.log
      - SIMPLETON_SERVER_LOG_PATH=/var/log/simpleton/server.log
      - SIMPLETON_CONTENT_PATH=/www
      - SIMPLETON_CHROOT=true
    restart: unless-stopped
```
