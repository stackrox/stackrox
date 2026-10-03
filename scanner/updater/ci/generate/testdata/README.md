# Independent matching inputs

`struts-packages.json` is the installed dpkg inventory (including source names
and versions) from the amd64 `quay.io/rhacs-eng/qa-multi-arch:struts-app` image.
The locally cached image has repository digest
`sha256:3401cc09305901ed505ddfccf3e533dde3f7fe326958131ee653d4067ab32246`.
Its `/usr/local/tomcat/webapps/ROOT.war` contains
`WEB-INF/lib/struts2-core-2.3.12.jar`.

The database-backed regression test builds an index report from this inventory
and that Maven package. This is a focused matching test, not a substitute for
indexing the full image corpus. The generator never reads this directory.
Expected findings and the >=138 threshold are maintained in test code, not in
the vulnerability fixtures.
