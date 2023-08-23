# 0003 - Scanner image names

- **Author(s):** Ross Tannenbaum
- **Created:** 2023-08-22

## Status

Accepted.

## Context

The [current version of StackRox Scanner](https://github.com/stackrox/scanner) shipped with each StackRox release is a fork of [Clair v2](https://github.com/quay/clair/tree/v2.1.8) which is versioned and released separately from the rest of StackRox.
The separate versioning process has proven to cause confusion in the past, so we re-tag each Scanner release with the associated StackRox release.
For example, the Scanner 2.30.3 release triggered the creation of the following images:

* scanner:2.30.3
* scanner-slim:2.30.3
* scanner-db:2.30.3
* scanner-db-slim:2.30.3

Upon cutting the related StackRox release, 4.1.3, the images were re-tagged as follows:

* `scanner:4.1.3`
* `scanner-slim:4.1.3`
* `scanner-db:4.1.3`
* `scanner-db-slim:4.1.3`

There is an ongoing effort to overhaul the StackRox Scanner to align it more closely with [Clair v4](https://github.com/quay/clair).
To align with Clair's versioning scheme, we refer to the new StackRox Scanner as StackRox Scanner v4.
StackRox Scanner v4 is housed in this repository (as opposed to the previous, separate repository) and its release process is more tightly coupled with the StackRox release process.
So, the Scanner v4 images will always be tagged the same as the [main image](https://quay.io/repository/stackrox-io/main).

There will be a period of time when StackRox will ship both the older Scanner, Scanner v2, and the new Scanner, Scanner v4.
It is clear that both versions of StackRox Scanner cannot have the same name and tag, so we must decide on a new name for either or both scanners.

## Decision

The new StackRox Scanner v4 images will be named as follows:

* `scannerv4`
* `scannerv4-db`

Note: StackRox Scanner v4 does not have "slim" versions.

## Other Considerations

### Renaming StackRox Scanner v2 images

We have considered renaming the older Scanner images to avoid adding versioning information to new Scanner images.
The new name for Scanner v2 was proposed to be `scanner-classic`.
This is nice for two reasons:

1. Avoids the need to rename future major Scanner versions (ie Scanner v5, if it were to be created)
2. Avoids adding Scanner versioning information in the name which may cause confusion
    * Recall the purpose of re-tagging Scanner images now is to avoid versioning confusion. The fact that StackRox is on version 4.x and Scanner is on version 4.x is completely coincidental. It is very possible Scanner goes through another overhaul while StackRox is still on version 4.x (or vice versa).

However, renaming Scanner v2 images comes with its own risks:

* It is easy for devs to make a mistake when testing
    * StackRox CI still relies heavily on Scanner v2. We do not wish to interfere with other team's workflows by forgetting to update the image name somewhere in CI.
    * StackRox devs still rely heavily on Scanner v2. We do not wish to interfere with their workflows by forgetting to update the image name somewhere in deployment scripts.
* It is easy for users to make a mistake when upgrading the Scanner image
    * Some users upgrade manually via `roxctl`. It could be easy to miss the new name for Scanner v2 and for users to run into an issue.
