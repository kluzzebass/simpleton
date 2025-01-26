# simpleton
Simpleton HTTPd is a really simple web server that serves static content from a directory.







## Docker

```
docker build -t kluzz/simpleton:latest .
docker run -p 80:80 -v /tmp/simpleton/content:/www kluzz/simpleton:latest
```