# Tag metadata cache

`tagfetcher` derives metadata for a tag from the registry and the watcher
persists it in `base_image_tags`. A repository scan calls the registry for every
checked tag, but the result is only written when the cache entry is not current.

## Freshness rule

A cache entry is current when **both** hold:

1. the digest fetched from the registry equals the cached `manifest_digest`;
2. the cached `metadata_version` equals `tagfetcher.CurrentMetadataVersion`.

Otherwise the fetcher emits a metadata event and the watcher upserts the row.

## Changing cached fields

`CurrentMetadataVersion` versions the **derivation**, not the proto schema. Whenever a
field that is persisted in the cache is added, removed, or computed
differently, bump the constant. No data migration is required:

- existing rows carry the proto3 zero value or an older number, so they fail
  the freshness check;
- on their next normal scan the fetched metadata is persisted and stamped with
  the new version;
- rows that are never checked (for example tags outside the top-N) stay stale
  until they are checked, which is harmless because nothing reads them.

Version history:

| Version | Change |
|---|---|
| 1 | Added `BaseImageTag.history` |

## Rules

- The constant is owned by the producer of the derived data, not by the store.
- The producer stamps every write with the current version; consumers never
  rewrite cached payloads to invalidate them.
- Never reuse a version number, and bump for derivation changes even when the
  proto is unchanged.
- Zero is the "unknown/legacy" value by construction, so an upgrade invalidates
  old rows without a backfill.
