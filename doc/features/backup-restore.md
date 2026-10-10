# Backup and Restore

**Primary Packages**: `central/globaldb/v2backuprestore`, `central/externalbackups`, `roxctl`
**Components**: Backup manager, cloud storage plugins, restore process, scheduling

## Overview

Database backup to external cloud storage (AWS S3, S3-compatible, Google Cloud Storage) with automated scheduling, on-demand backups, and database restore. Plugin-based architecture for extensibility, handles complete database export and import pipeline.

**Capabilities**: Automated scheduled backups via cron (daily/weekly/monthly), on-demand execution, PostgreSQL native format, cloud storage integration (AWS S3, S3-compatible, GCS), database restore, retention management, integration health monitoring, certificate backup/restore for disaster recovery.

## Architecture

**Manager** at `central/globaldb/v2backuprestore/manager/`: Orchestrates operations via Manager interface (GetExportFormats, GetSupportedFileEncodings, LaunchRestoreProcess, GetActiveRestoreProcess). Responsibilities: plugin instantiation from config, schedule management via scheduler, backup execution coordination, concurrency control (prevents simultaneous backups). Concurrency control uses `managerImpl` with lock mutex and inProgress flag allowing single backup at a time.

**Scheduler** at `central/externalbackups/scheduler/`: Manages cron-based schedules via Scheduler interface (UpsertBackup, RemoveBackup, RunBackup). Cron specs: `0 2 * * *` (daily 2 AM), `0 0 * * 0` (weekly Sunday), `0 0 1 * *` (monthly 1st), aliases `@daily`, `@weekly`. Backup pipeline: creates pipe reader/writer, exports DB to pipe writer in goroutine, streams to cloud storage, updates integration health.

### Plugin Architecture

**Plugin Interface** at `central/externalbackups/plugins/types/`: ExternalBackup interface with Backup(reader) and Test() methods.

**Registry** at `central/externalbackups/plugins/`: Map of type string to CreatorFunc with entries for s3, s3compatible, gcs. Registration uses init pattern calling `plugins.Registry["s3"] = NewS3`.

**S3 Plugin** at `central/externalbackups/plugins/s3/std/`: AWS S3 using SDK v2. Config: Bucket, UseIam, AccessKeyId, SecretAccessKey (encrypted), Region, ObjectPrefix, Endpoint. Features: IAM role auth (recommended for EKS/EC2), static credentials support, custom endpoint override, object prefix for organizing backups, server-side encryption, multi-part upload for large backups. Migration (ROX-22597): migrated from AWS SDK v1 to v2 with improved error handling and retry logic, better performance with streaming uploads.

**S3-Compatible Plugin** at `central/externalbackups/plugins/s3/compatible/`: Supports MinIO, Ceph, OpenStack Swift. Additional config: custom endpoint required, path-style vs virtual-hosted-style addressing, checksum validation (ROX-26945). Differences from AWS S3: no IAM role support (requires static credentials), endpoint must be specified, some S3 features may not be supported (versioning, lifecycle).

**GCS Plugin** at `central/externalbackups/plugins/gcs/`: Google Cloud Storage. Config: Bucket, ServiceAccount (JSON key file), ObjectPrefix. Features: service account authentication, customer-managed encryption keys (CMEK) support, uniform bucket-level access, signed URL support for restore.

**PostgreSQL v1 Format** at `central/globaldb/v2backuprestore/formats/postgresv1/`: Uses pg_dump for database export, includes Central certificates (CA bundles), supports gzip compression, manifest-based file tracking. File structure: backup.zip contains manifest.json, db/postgres-dump.sql.gz, cas/ca.pem|ca-key.pem|jwt-key.pem. Manifest has formatVersion, centralVersion, timestamp, files with encoding and sha256.

## Export Pipeline

Steps: (1) acquire snapshot (read-consistent view), (2) export schema (CREATE TABLE statements), (3) export data (rows in batches using COPY or SELECT), (4) stream to pipe (io.PipeWriter for streaming upload), (5) compression (gzip applied by export module), (6) encryption (TLS in transit, server-side at rest).

Certificate inclusion: by default backups include Central's TLS certificates. Controlled by `plugin.GetIncludeCertificatesOpt()` and `plugin.GetIncludeCertificates()`. Include for full disaster recovery (restore Central with same identity), exclude for data portability (certificates regenerated on restore).

## Restore Process

**Lifecycle** at `central/globaldb/v2backuprestore/manager/restore_process.go`: RestoreProcess interface with Metadata and Completion methods. Steps: (1) validate manifest and format, (2) create temporary restore directory, (3) launch restore process (single active allowed), (4) extract and decompress files, (5) run format-specific restore handlers, (6) on success restart Central to pick up new DB, (7) on failure cleanup and report error.

Concurrency control: only one active restore at a time checked via `m.activeProcess != nil`. Automatic restart: on successful restore Central restarts itself via `osutils.Restart()` after grace period.

PostgreSQL restore process: stop all Central services, drop existing database, restore from pg_dump file, restore certificates if included, validate schema, restart services. Limitations: restore not supported with external databases (ROX-18005), cannot restore databases from before version 4.0 (ROX-26376).

## Data Flow

**Scheduled Backup**: Cron Trigger → Scheduler.backupClosure → Scheduler.RunBackup → Export.BackupPostgres → Pipe Writer → Plugin.Backup(reader) → Cloud Storage → Update Integration Health.

**On-Demand Backup**: User → Service.PostBackup → Manager.Backup(id) → Check inProgress flag → Scheduler.RunBackup → (same as scheduled).

**Config Update**: User → Service.PutExternalBackup → Manager.Upsert → DataStore.UpdateExternalBackup → Scheduler.UpsertBackup (update cron) → Plugin re-instantiation.

**Database Restore**: User → Service.RestoreDB → Manager.LaunchRestoreProcess → Validate manifest → Extract backup files → Run restore handlers → Restart Central (on success).

## roxctl Integration

Backup commands: `roxctl central backup --endpoint central.example.com:443 --token-file /path/to/token --output backup-$(date +%Y%m%d).zip` or `roxctl central db backup --endpoint central.example.com:443 --output backup.zip` (v2).

Restore commands: `roxctl central db restore --endpoint central.example.com:443 --file backup.zip` (v2). Note: requires admin privileges and works only with embedded PostgreSQL.

## Services

**ExternalBackupsService** at `central/externalbackups/service/`: gRPC endpoints GetExternalBackups, GetExternalBackup(id), PostExternalBackup, PutExternalBackup, DeleteExternalBackup(id), TestExternalBackup, TestUpdatedExternalBackup.

**DBService** at `central/globaldb/v2backuprestore/service/`: gRPC methods for database export/backup, restore metadata queries. HTTP handlers: POST /v1/db/restore (initiate), POST /v1/db/restore/resume (resume interrupted). Authorization requires admin access.

## Code Locations

**Core**: `central/globaldb/v2backuprestore/manager/manager.go`, `manager/restore_process.go`, `service/service_impl.go`.

**Formats**: `central/globaldb/v2backuprestore/formats/registry.go`, `formats/postgresv1/`.

**Backup Generators**: `central/globaldb/v2backuprestore/backup/generators/dbs/` (DB generators), `generators/cas/` (CA generators), `generators/stream.go` (stream generator), `generators/zip.go` (ZIP generator).

**External Backups**: `central/externalbackups/manager/manager.go`, `scheduler/schedule.go`, `datastore/datastore_impl.go`, `plugins/` (s3/std/, s3/compatible/, gcs/).

**Database Export**: `central/globaldb/export/backup.go`.

## Environment Variables

**Storage**: `STORAGE=pvc` enables persistent storage for PostgreSQL (recommended for backups).

**External Backups**: Plugin configurations stored in database, cloud credentials encrypted at rest.

## Recent Changes

ROX-30629 (2024) separated external backup and external DB, split backup service from database restore service for better separation of concerns. ROX-26945 (2023) increased backup test timeouts, addressed flaky tests with large databases, improved reliability. ROX-24916 (2023) added prune option in store, cleanup of old backups via retention policy, configurable via `backups_to_keep`. ROX-22597 (2023) modernized S3 plugin, migrated to AWS SDK v2, extracted common S3 code for reuse, improved error handling and retry logic. ROX-22019 (2022) migrated to protobuf-go v2 for better performance and API consistency. ROX-18005 (2024) cleaned up external database support, removed restore to external databases, simplified to embedded PostgreSQL only.

## Best Practices

**Backup Strategy**: Schedule regular backups at off-peak hours, periodically test restores to staging, set `backups_to_keep` to balance storage cost vs retention needs, configure backups to multiple cloud providers for redundancy, keep `include_certificates` enabled for full disaster recovery, set up alerts for backup integration health status.

**Cloud Storage**: Use IAM roles instead of static credentials when possible (AWS EKS), configure cloud storage lifecycle policies for old backups, enable server-side encryption on S3/GCS buckets, restrict bucket access to minimum required permissions, enable bucket versioning for protection against accidental deletion.

**Restore Operations**: Always create backup before restoring (extra safety), test restores in non-production first, plan for Central downtime during restore, ensure backup version compatible (4.0+), understand certificate implications (include/exclude decision).

## Performance

**Backup Performance**: Snapshot acquisition may briefly block writes, large exports can saturate disk I/O, uploads compete with normal traffic, streaming design minimizes memory footprint, gzip compression reduces transfer size but adds CPU load.

**Optimization**: Schedule backups during low-traffic periods (e.g., 2 AM), use compression for large databases, configure appropriate retention to limit storage costs, monitor backup duration and adjust as database grows.

## Troubleshooting

**S3 Connection Issues**: Verify AWS credentials or IAM role permissions, check S3 bucket exists and accessible, verify region configuration, test with `TestExternalBackup()` API.

**GCS Connection Issues**: Verify service account JSON key valid, check GCS bucket permissions, ensure bucket exists in correct project, test with `TestExternalBackup()` API.

**Backup Size Issues**: Check available disk space on Central node, verify cloud storage bucket has sufficient quota, consider multi-part upload limits for large databases.

**Restore Failures - Version Incompatibility**: Error "restoration prior to 4.0 is not supported" means cannot restore pre-4.0 backups, must migrate manually.

**Restore Failures - External Database**: Error "restore is not supported with external database" means use native database tools (pg_restore) for external PostgreSQL.

**Restore Failures - Permissions**: Verify user has admin permissions, check Central has write access to database.

**Restore Failures - Active Process**: Only one restore can run at a time, wait for current restore to complete or fail.

## Security Considerations

**Credential Management**: Credentials encrypted in database, no integration with external secret managers (Vault, etc.) currently, credentials stored in database backup.

**Backup Contents**: Database includes all security policies, vulnerabilities, alerts, certificates included by default (full system identity), sensitive configuration data present.

**Access Control**: Backup configuration requires admin permissions, restore operations require admin access, cloud storage should have restricted access.
