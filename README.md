# cdn

CDN microservice to upload files to zachlatta.com that only accepts traffic from Tailscale IPs.

source code available at https://github.com/zachlatta/cdn

## config

the following env variables must be set:

- `FS_DEST_DIR` - path on the filesystem to store files (ex. "/files")
- `BASE_URL` - base URL to append filenames to when returning live URLs (ex. "https://zachlatta.com/f/")
- `ALLOWED_SUBNET` - subnet to accept requests from (default: tailscale)

## usage

POST file to "/upload" with "file" field name in multipart form

GET "/" -> return contents of README.md
