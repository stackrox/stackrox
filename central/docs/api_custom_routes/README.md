# Custom Route API specs
This directory includes manually created OpenAPI/Swagger specs for Central's custom routes defined in [/central/main.go](/central/main.go) (refer to `customRoutes()`)

These particular specs are currently NOT accessible via Central, instead they are/will be available in the official docs.

TODO(ROX-28173): improve doc creation, automation, serving, etc..

## Manually Modifying and Converting to AsciiDoc

_Disclaimer: Before executing these steps confirm with the docs team if the process, scripts, etc. is still accurate. These steps represent an initial PoC and can/should be improved, which is in scope for ROX-28173_

The OpenAPI specs must be converted to AsciiDoc for publishing to the official docs.

Sample steps to modify, validate, and convert these specs:

1. Load the `yaml` into https://editor.swagger.io/ (or use the equiv offline variation) and make appropriate changes and validate syntax

2. Copy the modified `yaml` back into this repo

3. Generate an AsciiDoc from the modified `yaml` (change `swagger_dir` and `srcfile` as appropriate)
          
       swagger_dir=/path-to-stackrox-repo/central/docs/api_custom_routes

       srcfile=image_service_swagger.yaml

       docker run --rm -v "$swagger_dir:/local" openapitools/openapi-generator-cli generate -i /local/$srcfile -g asciidoc -o /local/asciidoc

4. Clone `rhacs-api-docs-gen` repo: https://github.com/gaurav-nelson/rhacs-api-docs-gen

5. Move the generated `adoc` file to `scripts/` dir of `rhacs-api-docs-gen` tool

       mv $swagger_dir/asciidoc/index.adoc /path-to-rhacs-api-docs-gen-repo/scripts

6. Execute `scripts/updateasciidoc.js` to cleanup the `adoc` file (will be modified in place)

       cd /path-to-rhacs-api-docs-gen-repo

       node scripts/updateasciidoc.js index.adoc

7. Provide the new `adoc` file(s) to docs team for publishing

8. Commit/push/merge updated `yaml`