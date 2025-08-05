package cve

import "regexp"

// This is a slightly more relaxed version of the validation pattern in the NVD
// JSON schema: https://csrc.nist.gov/schema/nvd/api/2.0/source_api_json_2.0.schema
//
// It allows for "CVE" to be case-insensitive and for dashes and underscores
// between the different segments.
var Pattern = regexp.MustCompile(`(?i:cve)[-_][0-9]{4}[-_][0-9]{4,}`)
