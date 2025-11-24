# File-Service

## Run the service

- Run: `go run ./cmd/server`
- Fix dependencies: `go mod tidy`

## Handling huge numbers of files

To efficiently handle users with very large libraries, the ListFiles endpoint now supports server-side pagination.

- Endpoint: `GET /files`
- Query parameters:
  - `page` (optional, default: 1)
  - `pageSize` (optional, default: 50, max: 500)

Notes:
- Pagination avoids loading all file records into memory for users with huge libraries.
- Database has indexes on user_id and uploaded_at to support fast paginated queries.

If you operate at massive scale, also consider:
- Using a CDN or presigned URLs for downloads/previews.
- Asynchronous preview generation to keep upload latency low.
- Streaming uploads/downloads directly to/from object storage.