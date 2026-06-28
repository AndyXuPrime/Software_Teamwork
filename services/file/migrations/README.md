# File Service Migrations

No production PostgreSQL repository is implemented in this MVP.

When PostgreSQL metadata persistence is added, create forward-only migrations here. The expected first table should store file-owned metadata only:

- public document id
- knowledge base id
- owner user id
- display filename
- content type
- size bytes
- tags
- server-generated object key
- status visible to file-owned routes
- created and deleted timestamps

Do not store raw file contents in PostgreSQL and do not expose object keys through API responses.
