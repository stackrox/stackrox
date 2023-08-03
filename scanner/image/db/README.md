# ScannerDB Image

ScannerDB is a PostgreSQL-based image.

To see the version of PostgreSQL used, check PG_MAJOR in the Dockerfile.

The image requires a pg_hba.conf (client authentication configuration) file and postgresql.conf (server configuration) file.
These are specified via ConfigMap for the ScannerDB Kubernetes deployment.
