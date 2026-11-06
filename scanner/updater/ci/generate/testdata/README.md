# Struts matching input

`struts-packages.json` is the dpkg inventory used by the database-backed
`TestFixtureMatching/struts aggregate` case. It records the installed package,
source-package, architecture, and version fields from the amd64
`quay.io/rhacs-eng/qa-multi-arch:struts-app` image at digest
`sha256:3401cc09305901ed505ddfccf3e533dde3f7fe326958131ee653d4067ab32246`.
The test adds the image's `struts2-core-2.3.12.jar` Maven package separately.

This independent inventory lets the test check aggregate matches using the
package relationships from a real image. The expected finding threshold lives
in test code; the fixture generator does not read this file.
