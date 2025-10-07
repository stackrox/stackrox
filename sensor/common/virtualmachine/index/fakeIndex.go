package index

import (
	"fmt"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"google.golang.org/protobuf/encoding/protojson"
)

// IndexReport returns a *v4.IndexReport unmarshaled from IndexReportJSON.
// Panics on error so it can be used in tests and examples without boilerplate.
func getHardcodedIndexReport() *v4.IndexReport {
	var m v4.IndexReport
	if err := protojson.Unmarshal([]byte(Indexreportjson1539), &m); err != nil {
		panic(fmt.Errorf("unmarshal IndexReport: %w", err))
	}
	return &m
}

// Indexreportjson1539 is the canonical JSON used to build IndexReport_1539.
const Indexreportjson1539 = `{
  "contents": {
    "environments": {
      "0": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "1": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "10": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "100": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "101": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "102": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "103": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "104": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "105": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "106": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "107": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "108": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "109": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "11": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "110": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "111": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "112": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "113": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "114": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "115": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "116": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "117": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "118": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "119": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "12": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "120": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "121": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "122": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "123": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "124": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "125": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "126": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "127": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "128": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "129": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "13": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "130": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "131": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "132": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "133": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "134": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "135": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "136": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "137": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "138": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "139": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "14": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "140": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "141": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "142": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "143": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "144": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "145": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "146": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "147": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "148": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "149": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "15": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "150": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "151": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "152": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "153": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "154": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "155": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "156": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "157": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "158": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "159": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "16": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "160": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "161": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "162": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "163": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "164": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "165": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "166": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "167": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "168": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "169": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "17": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "170": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "171": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "172": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "173": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "174": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "175": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "176": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "177": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "178": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "179": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "18": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "180": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "181": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "182": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "183": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "184": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "185": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "186": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "187": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "188": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "189": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "19": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "190": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "191": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "192": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "193": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "194": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "195": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "196": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "197": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "198": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "199": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "2": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "20": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "200": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "201": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "202": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "203": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "204": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "205": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "206": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "207": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "208": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "209": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "21": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "210": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "211": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "212": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "213": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "214": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "215": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "216": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "217": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "218": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "219": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "22": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "220": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "221": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "222": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "223": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "224": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "225": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "226": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "227": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "228": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "229": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "23": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "230": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "231": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "232": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "233": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "234": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "235": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "236": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "237": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "238": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "239": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "24": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "240": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "241": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "242": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "243": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "244": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "245": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "246": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "247": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "248": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "249": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "25": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "250": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "251": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "252": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "253": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "254": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "255": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "256": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "257": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "258": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "259": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "26": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "260": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "261": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "262": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "263": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "264": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "265": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "266": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "267": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "268": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "269": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "27": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "270": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "271": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "272": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "273": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "274": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "275": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "276": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "277": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "278": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "279": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "28": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "280": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "281": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "282": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "283": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "284": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "285": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "286": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "287": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "288": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "289": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "29": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "290": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "291": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "292": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "293": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "294": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "295": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "296": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "297": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "298": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "299": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "3": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "30": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "300": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "301": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "302": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "303": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "304": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "305": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "306": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "307": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "308": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "309": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "31": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "310": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "311": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "312": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "313": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "314": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "315": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "316": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "317": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "318": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "319": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "32": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "320": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "321": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "322": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "323": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "324": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "325": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "326": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "327": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "328": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "329": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "33": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "330": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "331": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "332": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "333": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "334": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "335": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "336": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "337": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "338": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "339": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "34": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "340": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "341": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "342": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "343": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "344": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "345": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "346": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "347": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "348": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "349": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "35": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "350": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "351": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "352": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "353": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "354": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "355": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "356": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "357": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "358": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "359": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "36": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "360": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "361": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "362": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "363": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "364": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "365": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "366": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "367": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "368": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "369": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "37": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "370": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "371": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "372": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "373": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "374": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "375": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "376": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "377": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "378": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "379": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "38": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "380": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "381": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "382": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "383": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "384": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "385": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "386": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "387": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "388": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "389": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "39": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "390": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "391": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "392": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "393": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "394": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "395": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "396": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "397": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "398": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "399": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "4": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "40": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "400": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "401": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "402": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "403": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "404": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "405": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "406": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "407": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "408": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "409": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "41": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "410": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "411": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "412": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "413": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "414": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "415": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "416": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "417": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "418": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "419": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "42": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "420": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "421": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "422": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "423": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "424": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "425": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "426": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "427": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "428": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "429": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "43": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "430": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "431": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "432": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "433": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "434": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "435": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "436": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "437": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "438": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "439": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "44": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "440": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "441": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "442": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "443": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "444": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "445": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "446": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "447": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "448": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "449": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "45": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "450": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "451": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "452": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "453": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "454": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "455": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "456": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "457": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "458": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "459": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "46": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "460": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "461": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "462": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "463": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "464": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "465": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "466": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "467": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "468": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "469": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "47": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "470": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "471": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "472": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "473": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "474": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "475": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "476": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "477": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "478": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "479": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "48": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "480": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "481": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "482": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "483": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "484": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "485": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "486": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "487": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "488": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "489": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "49": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "490": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "491": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "492": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "493": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "494": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "495": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "496": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "497": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "498": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "499": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0"
            ]
          }
        ]
      },
      "5": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "50": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "500": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0"
            ]
          }
        ]
      },
      "501": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "502": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "503": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "504": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "505": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "506": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "1"
            ]
          }
        ]
      },
      "51": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "52": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "53": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "54": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "55": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "56": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "57": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "58": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "59": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "6": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "60": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "61": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "62": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "63": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "64": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "65": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "66": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "67": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "68": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "69": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "7": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "70": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "71": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "72": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "73": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "74": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "75": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "76": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "77": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "78": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "79": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "8": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "80": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "81": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "82": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "83": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "84": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "85": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "86": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "87": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "88": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "89": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "9": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "90": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "91": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "92": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "93": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "94": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "95": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "96": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "97": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "98": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      },
      "99": {
        "environments": [
          {
            "introducedIn": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
            "packageDb": "sqlite:var/lib/rpm",
            "repositoryIds": [
              "0",
              "1"
            ]
          }
        ]
      }
    },
    "packages": [
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "286",
        "kind": "binary",
        "name": "elfutils-debuginfod-client",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3857ce104171cf76afaa5b0cddaffd7baa06902773284121f05e714d1776143f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "elfutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.192-6.el9_6"
        },
        "version": "0.192-6.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "291",
        "kind": "binary",
        "name": "dracut-squash",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A26c596e4383f6f7c90084f1dd4c8262c98f638c2a7280b323ad34e0682d7ade9&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dracut",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "057-88.git20250311.el9_6"
        },
        "version": "057-88.git20250311.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "338",
        "kind": "binary",
        "name": "python3-audit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Afed6e687214c20e675d9c7d15a0bc1a53de982ab191829540b612ec32553c2c3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "audit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.1.5-4.el9"
        },
        "version": "3.1.5-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "370",
        "kind": "binary",
        "name": "NetworkManager-team",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abd17d96b886a180ada27f01dccb02a569fbe2f63ff06dacf4b90a285cadd9916",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "NetworkManager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.52.0-7.el9_6"
        },
        "version": "1:1.52.0-7.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "59",
        "kind": "binary",
        "name": "libacl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7eba3359b5b6de7dd39c59c9278df76256330d950a0ec1c325c794891b7a641f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "acl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.3.1-4.el9"
        },
        "version": "2.3.1-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "154",
        "kind": "binary",
        "name": "libsss_nss_idmap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abf0b66d900e21c40be3c57e629e87fc413a96fb0650ee1f5f71e67f3c5eb4429&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "209",
        "kind": "binary",
        "name": "libevent",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adc0460db752f3291f48930314db8427dbc97620e5443c821edc8ff81aede7761&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libevent",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.1.12-8.el9_4"
        },
        "version": "2.1.12-8.el9_4"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "281",
        "kind": "binary",
        "name": "systemd-udev",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5b5047edcfe9ca443b479253daaee50154ec653f366186a2394b4b023479c39d",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "systemd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "252-51.el9_6.2"
        },
        "version": "252-51.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "393",
        "kind": "binary",
        "name": "perl-base",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3431cfaad72ff70807cb56987d81be8b0648f38416e0876467f8cf544237568d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "2.27-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "216",
        "kind": "binary",
        "name": "systemd-pam",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae999f7b60fa5131bf0b7ef925da093c629f60e6c643096d2d2eff2de1ca174f7",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "systemd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "252-51.el9_6.2"
        },
        "version": "252-51.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "219",
        "kind": "binary",
        "name": "dbus-broker",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A554167db391921a548f0ab712140c67847c75534b24d998db190851eddbce1a1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dbus-broker",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "28-7.el9"
        },
        "version": "28-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "377",
        "kind": "binary",
        "name": "python3-netifaces",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adab48502ed4eae7814df5835a22762baddfb2f9db2ba582dcbc7ccecd0463844&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-netifaces",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.10.6-15.el9"
        },
        "version": "0.10.6-15.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "379",
        "kind": "binary",
        "name": "python3-prettytable",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7d1229fe51d22b21c0bd62a7fb1503d387221b7f9d3a0c80abaf0bbabc7a6bdb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-prettytable",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.7.2-27.el9"
        },
        "version": "0.7.2-27.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "5",
        "kind": "binary",
        "name": "abattis-cantarell-fonts",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae69d54de809a67ba544c6fe71fbaee16a0af9905e90f3275949d1510980c2108&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "abattis-cantarell-fonts",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.301-4.el9"
        },
        "version": "0.301-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "458",
        "kind": "binary",
        "name": "authselect-compat",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad7a4d1483a7a0128046a3a9c1ce0e0ed6ebb7884cf8b567f6e6b49249d6520d7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "authselect",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.6-3.el9"
        },
        "version": "1.2.6-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "483",
        "kind": "binary",
        "name": "system-reinstall-bootc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3b318fec4ba514d58fc552044dabfc138f3916a495009ddf9df632b6d89672c0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "bootc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.1.6-3.el9_6"
        },
        "version": "1.1.6-3.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "257",
        "kind": "binary",
        "name": "libgudev",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa91e3724fa98836e736015e584c2db50aad24c0c028469745ef994728cb66b01&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libgudev",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "237-1.el9"
        },
        "version": "237-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "85",
        "kind": "binary",
        "name": "libpsl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1b5c6e72d40bbe59d2657e9bd9b6d3d6eea49de158b64f4d04aedf315010b7de&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libpsl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.21.1-5.el9"
        },
        "version": "0.21.1-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "285",
        "kind": "binary",
        "name": "kernel-core",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8abd675f8810b2fc425c382b05d1ff6f08dccbba6045930ee3bbea74e79370c6",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.14.0-570.41.1.el9_6"
        },
        "version": "5.14.0-570.41.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "308",
        "kind": "binary",
        "name": "libdnf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A92ef42c60e981d29c39d6bd13da5a52a551da9ffb1c8e3919adcecf2efe82834&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libdnf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.69.0-13.el9"
        },
        "version": "0.69.0-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "486",
        "kind": "binary",
        "name": "lsscsi",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5b89bd4a925ebe53f0f403440024cc163890ebe615358783fac0c0dd95476e1a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lsscsi",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.32-6.el9"
        },
        "version": "0.32-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "221",
        "kind": "binary",
        "name": "iputils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A53fb716eb584fce7b172a81b8d34b8e05b47d4879a061930cfc923b8c85785ee&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "iputils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20210202-11.el9_6.1"
        },
        "version": "20210202-11.el9_6.1"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "228",
        "kind": "binary",
        "name": "initscripts-service",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0b4a9d48c27ed97568280d586d4dee55daf4994f975b471c2c4ca6d6d90da72a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "initscripts",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "10.11.8-4.el9"
        },
        "version": "10.11.8-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "255",
        "kind": "binary",
        "name": "librhsm",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A70dc5e6d938a23195b31f7a26dc7c3100092f639702ab247c8bf2674b37642b8&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "librhsm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.0.3-9.el9"
        },
        "version": "0.0.3-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "261",
        "kind": "binary",
        "name": "libsoup",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A980730b4a2898358a0424f7958ba41ef7abf8be85ac7fcdd7867ff2fc8c4c270&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libsoup",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.72.0-10.el9_6.2"
        },
        "version": "2.72.0-10.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "501",
        "kind": "binary",
        "name": "sos",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8871953c87c594ebb8ab70b70e03397bd9ac5e31b153d05e59909af0d6ecf8dd&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sos",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "4.10.0-4.el9_6"
        },
        "version": "4.10.0-4.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "25",
        "kind": "binary",
        "name": "ncurses-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A81d16163ff40a93ab8b2bb46b70404869201e88bbc90abc6c87d41fc8bb46b74&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ncurses",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.2-10.20210508.el9_6.2"
        },
        "version": "6.2-10.20210508.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "380",
        "kind": "binary",
        "name": "python3-pyrsistent",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af667aa129825ed11bdc198411c382bae1cef79d4367334e42482961d66ec441f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-pyrsistent",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.17.3-8.el9"
        },
        "version": "0.17.3-8.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "454",
        "kind": "binary",
        "name": "yum-utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7300d9394ab73b49414468f387851301077d0c79a8806a097963de920cdc9061&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dnf-plugins-core",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.3.0-20.el9"
        },
        "version": "4.3.0-20.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "170",
        "kind": "binary",
        "name": "ipcalc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A90e5fd4a755a70b8beff0a43628d6ede4381ca54e7034752b0b76f9be2304acb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ipcalc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.0-5.el9"
        },
        "version": "1.0.0-5.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "240",
        "kind": "binary",
        "name": "elfutils-default-yama-scope",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2fa3d40b0f57e830259ab947d0ad2328690f732fdc0d4f734d6fe517645d1149&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "elfutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.192-6.el9_6"
        },
        "version": "0.192-6.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "283",
        "kind": "binary",
        "name": "NetworkManager",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6a319c70d97db87b38ca377938fa4652089f5c0b5992d78d85b693e5367d405e",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "NetworkManager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.52.0-7.el9_6"
        },
        "version": "1:1.52.0-7.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "368",
        "kind": "binary",
        "name": "insights-client",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A828eef54aed8b304cbfd3f597ed9109d65f4699c82d288283182013713c6e9c2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "insights-client",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.2.8-1.el9"
        },
        "version": "3.2.8-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "444",
        "kind": "binary",
        "name": "libsss_certmap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8d2762b7dd70ceb0b2213868aef85cd0caca9885ddab1da89c0246a527345840&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "487",
        "kind": "binary",
        "name": "rootfiles",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2163957e58c6753cd1298b9a92b59a2c679b3dabf252d2905f1f7c5d3c28b035&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rootfiles",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.1-34.el9"
        },
        "version": "8.1-34.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "488",
        "kind": "binary",
        "name": "kernel-modules-core",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3348a2ad73bd402a6d4565e1c0ee348b8cd1bf84ddfc97f7a9848dbe488e16bf&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms&repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "5.14.0-570.49.1.el9_6"
        },
        "version": "5.14.0-570.49.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "238",
        "kind": "binary",
        "name": "kpartx",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A029dcae646f0a811693ead0219575b9fa95d34fe622c93cf8afd7de28162e77e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "device-mapper-multipath",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.8.7-35.el9_6.1"
        },
        "version": "0.8.7-35.el9_6.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "323",
        "kind": "binary",
        "name": "python3-dbus",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3b77dd40b5dcd1b6139f05339b4d509907c033abadc9abf8797cd288bc37f01f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dbus-python",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.18-2.el9"
        },
        "version": "1.2.18-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "61",
        "kind": "binary",
        "name": "libsmartcols",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac3884091af7f4d905e2a26f18a87deebf6fcad8e7aa30e9984a860ef33133c84&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "util-linux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.37.4-21.el9"
        },
        "version": "2.37.4-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "234",
        "kind": "binary",
        "name": "device-mapper-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac8a7433ad808a4bb8230212c84b9cdf03a4a38f2a22c55e794901b16325d15eb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lvm2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.03.28-6.el9"
        },
        "version": "9:1.02.202-6.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "321",
        "kind": "binary",
        "name": "python3-setuptools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0cf41fa126566d43bf7bc3548a078f7fe50d24cfbd21bbc4754ccf8ee92c63d4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-setuptools",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "53.0.0-13.el9_6.1"
        },
        "version": "53.0.0-13.el9_6.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "359",
        "kind": "binary",
        "name": "setroubleshoot-server",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aeaa8e764c12cf387a52401db5a83617f6d78af2b0e37fc4e86a5baca325cfe4b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "setroubleshoot",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.3.32-1.el9"
        },
        "version": "3.3.32-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "415",
        "kind": "binary",
        "name": "perl-Symbol",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1cc12904a449c02b27ffd7667ad45f019dd0aab90eeeb85a87da86231ab02fef&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.08-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "462",
        "kind": "binary",
        "name": "grub2-pc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5958ca45dddb06a2ba71502703adce00c8750449e37eb81ce8f539c135176596&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grub2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.06-104.el9_6"
        },
        "version": "1:2.06-104.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "106",
        "kind": "binary",
        "name": "grep",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0fe884f7b13739ffc312a66cb727051aa0ff716577e23e9c2c1fb529b9cf2fde&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grep",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-5.el9"
        },
        "version": "3.6-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "340",
        "kind": "binary",
        "name": "python3-libsemanage",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Add3b62b7a90fad213ef4907b81d59b4dbde354358887d7d3e006dae21534919c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libsemanage",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-5.el9_6"
        },
        "version": "3.6-5.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "500",
        "kind": "binary",
        "name": "libappstream-glib",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad7251ebe88f36866f00654bed1f72d65ecff1f96ea08c2b00b4b6c5a7983a96c&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-appstream-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libappstream-glib",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms",
          "version": "0.7.18-5.el9_4"
        },
        "version": "0.7.18-5.el9_4"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "126",
        "kind": "binary",
        "name": "libss",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1ae896ab9459ec51366cb7443faea15527d12ec0cfcef04022b65b242d79d0a4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "e2fsprogs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.46.5-7.el9"
        },
        "version": "1.46.5-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "144",
        "kind": "binary",
        "name": "libdb",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8ab3224c0b8fe1619e1f540f0d74ef98480e417060df1a14c41c458662d28257&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libdb",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.3.28-57.el9_6"
        },
        "version": "5.3.28-57.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "342",
        "kind": "binary",
        "name": "python3-linux-procfs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A429667f5ef677e05f05a38c0482bb8770d7c3decb857134feca3c82c5015e0c8&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-linux-procfs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.7.3-1.el9"
        },
        "version": "0.7.3-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "386",
        "kind": "binary",
        "name": "perl-Digest",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1e9505e7a7b9d2de885d932bd5dcec75b4bfcfef823389817743b2f5208fd695&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Digest",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.19-4.el9"
        },
        "version": "1.19-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "490",
        "kind": "binary",
        "name": "p11-kit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A47fffa771ec0e6a78e631479affc16dcf3b6b513cf2eb709e2c17ec8ad490bea&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "p11-kit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "0.25.3-3.el9_5"
        },
        "version": "0.25.3-3.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "71",
        "kind": "binary",
        "name": "file-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8b70b96fbbd61f148700206c9cbe3d0700b29f95f3b305195db71fa2df3638e1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "file",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.39-16.el9"
        },
        "version": "5.39-16.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "105",
        "kind": "binary",
        "name": "pcre",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A01c525ebcdbd69cafc831211c8a3d26e0a5ae355a377dd4d8a72b055471a4135&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pcre",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.44-4.el9"
        },
        "version": "8.44-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "271",
        "kind": "binary",
        "name": "libuser",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad538ec1ca679178e5a9ec646ff0a39a207451647ab048b974ad74780d74a697d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libuser",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.63-16.el9"
        },
        "version": "0.63-16.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "326",
        "kind": "binary",
        "name": "python3-pyyaml",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A34b3b3f57782df3197f3178926fc7e6b43e41fb83e1a9f38aebaa04e98a28f06&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "PyYAML",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.4.1-6.el9"
        },
        "version": "5.4.1-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "388",
        "kind": "binary",
        "name": "perl-B",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A617eff67b436783a09be4337ab9ef8faa482b4685aa64cf163f1d7706f10632f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.80-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "50",
        "kind": "binary",
        "name": "expat",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1ed78b222d236e2f79eec1148476c49d90f4578ebfe97bc7583fbaac8b48abf1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "expat",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5.0-5.el9_6"
        },
        "version": "2.5.0-5.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "4",
        "kind": "binary",
        "name": "adobe-source-code-pro-fonts",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab26371cbf0c8d34aa313e021cf90f27218a1ada16ff48d5cc67becf52f0d478b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "adobe-source-code-pro-fonts",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.030.1.050-12.el9.1"
        },
        "version": "2.030.1.050-12.el9.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "42",
        "kind": "binary",
        "name": "keyutils-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A759bcdcff820c65b2954f9d506f4f8497d1f2dfb7f5a4a050e340701619f4ffa&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "keyutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.6.3-1.el9"
        },
        "version": "1.6.3-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "166",
        "kind": "binary",
        "name": "libestr",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abd1ed36503535e039b6b4677474d31258d901b37039cdfa5f8003e4913067532&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libestr",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.1.11-4.el9"
        },
        "version": "0.1.11-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "376",
        "kind": "binary",
        "name": "python3-markupsafe",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3978c1dd572365dc04647e15bcb69d0cb87107fa84971bf02cbe42ddd7150b51&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-markupsafe",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.1.1-12.el9"
        },
        "version": "1.1.1-12.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "416",
        "kind": "binary",
        "name": "perl-File-stat",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7126d6f8f4dded9e662d7d8fc6c1699fcc8ff8aeea00970763d92d3896a5a4a5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.09-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "83",
        "kind": "binary",
        "name": "which",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A086bd15b84ce4d3a8b7c0c81e34ed09d1cd0b3ce0cfbc14c8f06da10b9fce4a2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "which",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.21-30.el9_6"
        },
        "version": "2.21-30.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "267",
        "kind": "binary",
        "name": "openldap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af5efdfbb6326e6d2b9b314f211c11c22c3c36eee0323606639c794953505ed6d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openldap",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.6.8-4.el9"
        },
        "version": "2.6.8-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "390",
        "kind": "binary",
        "name": "perl-Data-Dumper",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4d4da3eb6e14b27965ec3aca572ac5b46bbb91dde3ddce64a49d037d5d59845e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Data-Dumper",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.174-462.el9"
        },
        "version": "2.174-462.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "406",
        "kind": "binary",
        "name": "perl-Term-ANSIColor",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad441b9193bf2b33f0b1ad3ee0ee37e3bd45d83d14d5a80c91358aba4f61335e2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Term-ANSIColor",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.01-461.el9"
        },
        "version": "5.01-461.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "423",
        "kind": "binary",
        "name": "perl-overloading",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9938292656e647b7fb79424e0ff949f1165d63bd5c61cbc7cc93906b694cb2d8&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "0.02-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "36",
        "kind": "binary",
        "name": "libxml2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac84fad89b577ad3ca092f92471e7b4d1c71fd7bd8fc94a226fbc103ead94ccea",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libxml2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.13-12.el9_6"
        },
        "version": "2.9.13-12.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "113",
        "kind": "binary",
        "name": "libutempter",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af2724dc1fa106f8d5ec1a053fc87f6b6feffa81c7b849849ac083c7abffe60f5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libutempter",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.1-6.el9"
        },
        "version": "1.2.1-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "252",
        "kind": "binary",
        "name": "pciutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac6e06c064cba44d5408b60c49ea617e7559a953508747d25e51bce5db90ed271&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pciutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.7.0-7.el9"
        },
        "version": "3.7.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "259",
        "kind": "binary",
        "name": "libproxy-webkitgtk4",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa10023706b226bdd6c742fbe98ff2576de73ea657c0f25247345acb8da34fd91&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libproxy",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.4.15-35.el9"
        },
        "version": "0.4.15-35.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "405",
        "kind": "binary",
        "name": "perl-POSIX",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6c9dd1357bbaf23f5e7e31cdf5318af5afeb1e9f61990bdaf651d36f73c70433&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.94-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "502",
        "kind": "binary",
        "name": "irqbalance",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3021eb3d9865ef8cbc8d5f19a4779520b646ce413893d67489303c4e9c1cebb5&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "irqbalance",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "1.9.4-2.el9_6.2"
        },
        "version": "2:1.9.4-2.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "97",
        "kind": "binary",
        "name": "libdhash",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6625cfe17321f524093a10a222a4d19323b1774d581f0a8ef1d111f97502ef22&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ding-libs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.1-53.el9"
        },
        "version": "0.5.0-53.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "119",
        "kind": "binary",
        "name": "attr",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A365b724bb207c1e99af739e56f552ad8c786890bb042e04fd056598fbe7a360e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "attr",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5.1-3.el9"
        },
        "version": "2.5.1-3.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "397",
        "kind": "binary",
        "name": "perl-Time-Local",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aeb8fec3d66e211c7eb993d52755cac657b7ed4b71d995030a4de14f4f81d2e7d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Time-Local",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.300-7.el9"
        },
        "version": "2:1.300-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "65",
        "kind": "binary",
        "name": "libsemanage",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad19ec44af86facbbbcded833d47935e759c9e472969f5b3f36cf02ba8780927b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libsemanage",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-5.el9_6"
        },
        "version": "3.6-5.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "224",
        "kind": "binary",
        "name": "dhcp-client",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abbc6747775ceb48f0267f69ea68f73451d6c4f20b178fa3d7960d62b73a6d011&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dhcp",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.4.2-19.b1.el9"
        },
        "version": "12:4.4.2-19.b1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "366",
        "kind": "binary",
        "name": "python3-subscription-manager-rhsm",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa90a0efacd0bd719732d72c2016ea214e824d75131b45bfb56985f471e9674b1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "subscription-manager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.29.45.1-1.el9_6"
        },
        "version": "1.29.45.1-1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "495",
        "kind": "binary",
        "name": "kernel-tools-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2b5b9b32310462f34dbe1fac82b4ac70a4f64ee06ec367e1820a83e65abd28d7&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms&repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "5.14.0-570.49.1.el9_6"
        },
        "version": "5.14.0-570.49.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "38",
        "kind": "binary",
        "name": "elfutils-libelf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A339a0454d71d4e7b4208754baa29c2382d6c40a6c5d67eaaa2ed0798be44f937&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "elfutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.192-6.el9_6"
        },
        "version": "0.192-6.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "339",
        "kind": "binary",
        "name": "python3-file-magic",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Acb8bac4352690f892a6e6ee8cdcb021cce1bd22208622ed581a2362e9fd4dc3a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "file",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.39-16.el9"
        },
        "version": "5.39-16.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "429",
        "kind": "binary",
        "name": "perl-Scalar-List-Utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A535cd3e1f05c4cf6c2ff3d562db853b9d4600232a231a4188fb96701e9afb5fe&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Scalar-List-Utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.56-462.el9"
        },
        "version": "4:1.56-462.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "447",
        "kind": "binary",
        "name": "sscg",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1b0d5222139499a98825b8419c450dd7dd99a7c13e1e3f9916e34cd067c5e2b5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sscg",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.0.0-7.el9"
        },
        "version": "3.0.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "96",
        "kind": "binary",
        "name": "libcollection",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A03506626ee3abf72d2a67b6730beebf41fc714b23218d1a5fd203fbb9f57119f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ding-libs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.1-53.el9"
        },
        "version": "0.7.0-53.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "237",
        "kind": "binary",
        "name": "cryptsetup-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adb50fbff62f6c9e2102704803e1cf387781f66f3ce33d00189ea95766c4ac37c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cryptsetup",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.7.2-3.el9_6.1"
        },
        "version": "2.7.2-3.el9_6.1"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "401",
        "kind": "binary",
        "name": "perl-Pod-Escapes",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abf07cbb23d2451d668115536dc5fe8f4400111804207f1d672906a6aed6699f7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Pod-Escapes",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.07-460.el9"
        },
        "version": "1:1.07-460.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "446",
        "kind": "binary",
        "name": "mokutil",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6e8c886a96785a550153b539ba7ef4d2e011bca4dcd35c2f6b91bcf522036b99&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "mokutil",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.0-4.el9"
        },
        "version": "2:0.6.0-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "494",
        "kind": "binary",
        "name": "kernel-modules",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ace18bb1afe0d6c9cb8dd4ff1e8f791814968413c3edc84770b1f34917885d86b&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms&repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "5.14.0-570.49.1.el9_6"
        },
        "version": "5.14.0-570.49.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "496",
        "kind": "binary",
        "name": "kernel-tools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5657278b3028b0b5a13bf91e88da78a2048db4d88c2888e1dfba61591e21c1a8&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms&repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "5.14.0-570.49.1.el9_6"
        },
        "version": "5.14.0-570.49.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "183",
        "kind": "binary",
        "name": "openssl-fips-provider",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A93dea91d1e439c769429f02dec8f8aff3ef38ea473333ffcc19a2de94d4483a0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openssl-fips-provider",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.0.7-6.el9_5"
        },
        "version": "3.0.7-6.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "279",
        "kind": "binary",
        "name": "librepo",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0bdf53a4924e8b5b2397575b84f4903ba3cdb15d6a9dfa22d1302f1fa16aaf2f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "librepo",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.14.5-2.el9"
        },
        "version": "1.14.5-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "439",
        "kind": "binary",
        "name": "perl-PathTools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9769ce196959ce0464cb53a06df4677ec43b6acb0cf637ada32d43eb8b8239bc&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-PathTools",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.78-461.el9"
        },
        "version": "3.78-461.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "15",
        "kind": "binary",
        "name": "basesystem",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aaaada29e7a9cab643ee3cc4eee876ea240668b776a129e9787b275f57c1e91d5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "basesystem",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "11-13.el9"
        },
        "version": "11-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "306",
        "kind": "binary",
        "name": "libmodulemd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Affc9e971a87a01981cfc5c97929b8e2ca6ee37fd08402487bd053f7d597e7474&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libmodulemd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.13.0-2.el9"
        },
        "version": "2.13.0-2.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "330",
        "kind": "binary",
        "name": "python3-iniparse",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab50693d8f728f7810f21dc7e690278184629539a88fea864c28adac86e05bf8d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-iniparse",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.4-45.el9"
        },
        "version": "0.4-45.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "467",
        "kind": "binary",
        "name": "parted",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1b3c97260ca405a1c1883abd46e66b2f2d493fd9aa20f7371e1563064ca3f71f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "parted",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.5-3.el9"
        },
        "version": "3.5-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "26",
        "kind": "binary",
        "name": "bash",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A034b5216359462b00c54a5b58f87b0b70410d4d70670cd50a82e686bca4cdaaa&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "bash",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.1.8-9.el9"
        },
        "version": "5.1.8-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "57",
        "kind": "binary",
        "name": "gdbm-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abddff3e446f33b1e53d2d6f33be3684eef62d66a7369f72d9a4681bab68134f8&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gdbm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.23-1.el9"
        },
        "version": "1:1.23-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "149",
        "kind": "binary",
        "name": "libnfnetlink",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1d1d82d578236d28b67bc7eb8cc68db2a2edc5d209eb2861a1df344ae64d2644&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libnfnetlink",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.1-23.el9_5"
        },
        "version": "1.0.1-23.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "305",
        "kind": "binary",
        "name": "grubby",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A913ec8612798a0bbb8d8a00ce818d01b4199988f093f982fd606d87e15023dda&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grubby",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.40-64.el9"
        },
        "version": "8.40-64.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "433",
        "kind": "binary",
        "name": "perl-parent",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A00fff89cd61fa293ef7601461443f3bc1eb062323e827debea90fac07cbfe65b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-parent",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.238-460.el9"
        },
        "version": "1:0.238-460.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "453",
        "kind": "binary",
        "name": "rhc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac74bd11d229a9e236b22005258e3afa00a6ace72ee5a73c106df6f593169228e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rhc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.2.7-1.el9_6"
        },
        "version": "1:0.2.7-1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "40",
        "kind": "binary",
        "name": "audit-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Afbe65f0718f55a810a8109b4eaae66babb4f7cd750350b272f8d6fa933ac7432&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "audit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.1.5-4.el9"
        },
        "version": "3.1.5-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "51",
        "kind": "binary",
        "name": "json-c",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aaeba795d62ed75372971eec2c79fb99467efe140860c8bb2be3e1f8c38726087&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "json-c",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.14-11.el9"
        },
        "version": "0.14-11.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "60",
        "kind": "binary",
        "name": "libmnl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A52a6a7ecc2b7ae73db760bb6d2f3b2b3f8b4ab9e0e4e10cb1c4c977698d71ad0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libmnl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.4-16.el9_4"
        },
        "version": "1.0.4-16.el9_4"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "189",
        "kind": "binary",
        "name": "libblkid",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A88c5ed7e447f050502c6db74efe809b7bb94a9dc0d8f6967f3e42ed73e5d320f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "util-linux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.37.4-21.el9"
        },
        "version": "2.37.4-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "273",
        "kind": "binary",
        "name": "usermode",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aab8f569e4d94a391d642b53f95580783923cd86b3ae81771c2869ca56d5b2e59&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "usermode",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.114-6.el9"
        },
        "version": "1.114-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "348",
        "kind": "binary",
        "name": "python3-libcomps",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2bc3716cd966cab653b90ba16f7ee91b26b14c922c52c445beac931bcda8073a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libcomps",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.1.18-1.el9"
        },
        "version": "0.1.18-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "381",
        "kind": "binary",
        "name": "python3-jsonschema",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0ef4a13cea9e7f055a46a16437d6ed865a0cf5f8d8c616ba6915c1ea27b3148d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-jsonschema",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.2.0-13.el9"
        },
        "version": "3.2.0-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "87",
        "kind": "binary",
        "name": "keyutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A06f27293394831ca0c1c211a5ca13d6b5c7a5ebe6338b0400f6f6e341be9a272&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "keyutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.6.3-1.el9"
        },
        "version": "1.6.3-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "155",
        "kind": "binary",
        "name": "libsss_sudo",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adb30ebf2298198889376e37b3d36b312a2dc5c6c117ade5a3e4ddef9e0fa2247&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "236",
        "kind": "binary",
        "name": "grub2-tools-minimal",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aaa5d6c83f49f72bb0c9d7ff43b0a162bfa2fb80df49c255d615807be874ce34e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grub2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.06-104.el9_6"
        },
        "version": "1:2.06-104.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "292",
        "kind": "binary",
        "name": "libfido2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A63086ed88d4ab09f9a1c878bd372dcf9bffc1b36945308699fc1dcbe1dce19aa&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libfido2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.13.0-2.el9"
        },
        "version": "1.13.0-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "309",
        "kind": "binary",
        "name": "libdnf-plugin-subscription-manager",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3f52e5a1975d3a18d71c1d2b64d6271cec83ba4141258c1dd4952cd79e41eb56&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "subscription-manager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.29.45.1-1.el9_6"
        },
        "version": "1.29.45.1-1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "81",
        "kind": "binary",
        "name": "numactl-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0aea07c740c6358f1c9bc85529b73279f7bda6a5a2a5f21c14073afda4dd618d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "numactl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.0.19-1.el9"
        },
        "version": "2.0.19-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "138",
        "kind": "binary",
        "name": "cpio",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Acab7988bb18f4c5bd373f2b3fd9e9c8391dd6fb9ec85f8dd98896ac58084736c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cpio",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.13-16.el9"
        },
        "version": "2.13-16.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "258",
        "kind": "binary",
        "name": "webkit2gtk3-jsc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A499e31110a1c62d169018bb7f1675b6e9ddd01f887417bb8255b37653fbc207a",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "webkit2gtk3",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.48.5-1.el9_6"
        },
        "version": "2.48.5-1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "287",
        "kind": "binary",
        "name": "binutils-gold",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A19c8ed5803aa44c31ad90d266d0aa2478223eb5381eccaec6c7028f9fb36acdf&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "binutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.35.2-63.el9"
        },
        "version": "2.35.2-63.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "314",
        "kind": "binary",
        "name": "rpm-plugin-systemd-inhibit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Acf220042af08a982126f2d2a3c36134220e610b588c21ae3e6ad5cf2f3b887ae&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "417",
        "kind": "binary",
        "name": "perl-podlators",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9140dfbc0f2cfcdc47c5f779fb5583883851fb6e6e475ccd71bd405b8a822aaf&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-podlators",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.14-460.el9"
        },
        "version": "1:4.14-460.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "112",
        "kind": "binary",
        "name": "gawk",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A63f6960c10bb471f28d12679026c080819985c33aece0226ebf95c209d182fe2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gawk",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.1.0-6.el9"
        },
        "version": "5.1.0-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "146",
        "kind": "binary",
        "name": "libev",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Acdbd89d1628b4a5361acb27302932bf797552d44f71a65d74906dba1d12924b1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libev",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.33-6.el9"
        },
        "version": "4.33-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "202",
        "kind": "binary",
        "name": "openssl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A735b87b1db37b85d482f305d8ff44208210b3cb400b60a57184c1e71796a1b62&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openssl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.2.2-6.el9_5.1"
        },
        "version": "1:3.2.2-6.el9_5.1"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "331",
        "kind": "binary",
        "name": "python3-inotify",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A687f8e4fbca1a8dbe598709c337bb2ad37460b4fd8f91614d56f7f004c71a032&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-inotify",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.9.6-25.el9"
        },
        "version": "0.9.6-25.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "431",
        "kind": "binary",
        "name": "perl-Storable",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab185a633bea4cb3bef0e0828528b0e19cc5c5f607ccacce4ee7a4a2c4bc8a48e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Storable",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.21-460.el9"
        },
        "version": "1:3.21-460.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "140",
        "kind": "binary",
        "name": "hdparm",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A36cd5a793851538882e317165e808a930ff306082565780aba5271e1f1375938&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "hdparm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "9.62-2.el9"
        },
        "version": "9.62-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "278",
        "kind": "binary",
        "name": "libcurl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A708b5314538a2eab7978324e79641a158d00f3d2dbba02d8dc7af7ed297c5c65&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "curl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "7.76.1-31.el9_6.1"
        },
        "version": "7.76.1-31.el9_6.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "301",
        "kind": "binary",
        "name": "rpm-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A23f3b282cca2aff26b6e692a9fcf364456ed09f11200c3ac720a6295c05cfd5c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "434",
        "kind": "binary",
        "name": "perl-vars",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5150744b046f855d969bfe8b0cb1ecb2a021c1f68b677b16b1aee1af633fc58a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.05-481.1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "457",
        "kind": "binary",
        "name": "crypto-policies-scripts",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A97dda38e7582e02657cf1091cc6392bed9741e6394074d1fac87a9fa601db0e7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "crypto-policies",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20250128-1.git5269e22.el9"
        },
        "version": "20250128-1.git5269e22.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "82",
        "kind": "binary",
        "name": "pciutils-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Afbab19367d7f9f0107ca14a806525ed990af472c63ed1d0ca4afef3f4d7dc2f8&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pciutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.7.0-7.el9"
        },
        "version": "3.7.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "230",
        "kind": "binary",
        "name": "virt-what",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A649210fefb19584053a3c042407cf763fff43fc347d975e580e1b1908e08bb23&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "virt-what",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.27-1.el9"
        },
        "version": "1.27-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "302",
        "kind": "binary",
        "name": "policycoreutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2b5c8483d4adcc9bfee25174696c8b0b4014ee06a7ef577a606d617f05908142&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "policycoreutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-2.1.el9"
        },
        "version": "3.6-2.1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "33",
        "kind": "binary",
        "name": "libuuid",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A00412278b09c2c4e6ba72d42c85b777706ede91d6ea0f7f4c209e582e0ef77aa&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "util-linux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.37.4-21.el9"
        },
        "version": "2.37.4-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "67",
        "kind": "binary",
        "name": "findutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A796395f9f26ca2155f4152e4c2ee6a878bd253534eb4c97e1bef5c54c8618a69&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "findutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.8.0-7.el9"
        },
        "version": "1:4.8.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "139",
        "kind": "binary",
        "name": "diffutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6e45ef7cb2a8e637dfaf59442c4768dbc1e4714209bb454c6e29f0095e65ab2d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "diffutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.7-12.el9"
        },
        "version": "3.7-12.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "180",
        "kind": "binary",
        "name": "dhcp-common",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A17e0b2e09210e690ee3f43f7f8c7a7ae2573bb3572175db06b98c1ad13f9f060&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dhcp",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.4.2-19.b1.el9"
        },
        "version": "12:4.4.2-19.b1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "191",
        "kind": "binary",
        "name": "libmount",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6628315b3381a7b94d1c3e69fbc8c2e8ee80b829a131a5b19c4ea87c610b3cb6&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "util-linux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.37.4-21.el9"
        },
        "version": "2.37.4-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "196",
        "kind": "binary",
        "name": "shared-mime-info",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5ba5c16a7c0045e19895b8e9ca1464b8fa30b74851481235ae6d12eff980b264&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "shared-mime-info",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.1-5.el9"
        },
        "version": "2.1-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "199",
        "kind": "binary",
        "name": "libfdisk",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af37d94d536aa74457b74a30bc30b421b68a48e915e6cf284129df21eda834452&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "util-linux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.37.4-21.el9"
        },
        "version": "2.37.4-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "91",
        "kind": "binary",
        "name": "efivar-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A66c1cc52c25be497b327eebc346ee4233b3a9c649f5110cdacfa89e4510963a1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "efivar",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "38-3.el9"
        },
        "version": "38-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "239",
        "kind": "binary",
        "name": "xfsprogs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af28fd760c43f5c1b6e8279d0d4b2bf454e1743588ff7e7f3a93195ba8bdcb009&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "xfsprogs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.4.0-5.el9"
        },
        "version": "6.4.0-5.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "1",
        "kind": "binary",
        "name": "subscription-manager-rhsm-certificates",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Acf5b8ca57cf25570b679ec1bbb7bab98ee62c6212965c376e9c35d450f9b2ffd&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "subscription-manager-rhsm-certificates",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20220623-1.el9"
        },
        "version": "20220623-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "411",
        "kind": "binary",
        "name": "perl-Pod-Simple",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4ce33fe8ad5639e341359e5f37f7cbd392a4c5b8786875cdf095aaca38de3247&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Pod-Simple",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.42-4.el9"
        },
        "version": "1:3.42-4.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "430",
        "kind": "binary",
        "name": "perl-constant",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aea82571becf83f723e0a165caefccb96684dc1ecf43cb2d9bf7246377bc2ab67&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-constant",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.33-461.el9"
        },
        "version": "1.33-461.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "455",
        "kind": "binary",
        "name": "yum",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9ba60ee910dff207ca9cb120c48191460513ba1d3b15a40a425cd4d4f4a2e0bb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dnf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.14.0-25.el9"
        },
        "version": "4.14.0-25.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "107",
        "kind": "binary",
        "name": "slang",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab8c0c7efabf8cbcfcf52134c797eee057352a95c17ca68190e53b40afce883a9&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "slang",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.3.2-11.el9"
        },
        "version": "2.3.2-11.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "134",
        "kind": "binary",
        "name": "less",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af82daf8e32570ff8322034977fd8807ee14b3236184a6f179d6a8abda134385c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "less",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "590-5.el9"
        },
        "version": "590-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "247",
        "kind": "binary",
        "name": "oddjob",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad710baebf44f1f93791b7f9b1bc1fd4afd680e975ae511871d946fd9dea9122f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "oddjob",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.34.7-7.el9"
        },
        "version": "0.34.7-7.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "304",
        "kind": "binary",
        "name": "selinux-policy-targeted",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A492c50b77f8d7bcacc5a4a7aeaf2819ea4e9824faeae465177c4492368276d91&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "selinux-policy",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "38.1.53-5.el9_6"
        },
        "version": "38.1.53-5.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "307",
        "kind": "binary",
        "name": "libsolv",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aafb16017c704afd31b455c9d43fb385c4ecfff4dceddd2216b298dca06800c6b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libsolv",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.7.24-3.el9"
        },
        "version": "0.7.24-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "312",
        "kind": "binary",
        "name": "rpm-build-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa01307f481bf72fd1ba0c0651b0fdede5928c05e23d3092127448f79615a3fee&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "100",
        "kind": "binary",
        "name": "libseccomp",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae064b0644786f03af5a6ecd48f581beb6c9b63c5c59a429e9b8f6f39b67b79f7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libseccomp",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5.2-2.el9"
        },
        "version": "2.5.2-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "282",
        "kind": "binary",
        "name": "dracut",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac4d81b62129a08011989a5380a935948443fad2df4e38eb31f7717f2b4d8fcea&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dracut",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "057-88.git20250311.el9_6"
        },
        "version": "057-88.git20250311.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "367",
        "kind": "binary",
        "name": "subscription-manager",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2990ca85962cf2282c71119aecc3f9ff417d1a7675b2ef7364b8c4ad45b773c6&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "subscription-manager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.29.45.1-1.el9_6"
        },
        "version": "1.29.45.1-1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "374",
        "kind": "binary",
        "name": "python3-jsonpointer",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A23978f42bb37784f0f457d14485aed44f89a05bc69af54323f7f50a34cf3c4c9&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-jsonpointer",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.0-4.el9"
        },
        "version": "2.0-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "371",
        "kind": "binary",
        "name": "PackageKit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A387b36e0189d792ae98c54434d0f922364022f5f01327592ea322b15c7e2b496&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "PackageKit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.6-1.el9"
        },
        "version": "1.2.6-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "491",
        "kind": "binary",
        "name": "libgcc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A05c4df5009add1eb26716b3135624ba960e5d68fdbc9465b49c889d07c43fbb8&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gcc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "11.5.0-5.el9_5"
        },
        "version": "11.5.0-5.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "24",
        "kind": "binary",
        "name": "glibc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aeb713887025c11cc5b448e616b1dc82976b8eb93f630a799173c9957a6418593",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "glibc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.34-168.el9_6.23"
        },
        "version": "2.34-168.el9_6.23"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "241",
        "kind": "binary",
        "name": "elfutils-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab963e8915ea2a6b05b52591f30df93907c29437c792cd9eaf4b4b29142543bc7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "elfutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.192-6.el9_6"
        },
        "version": "0.192-6.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "327",
        "kind": "binary",
        "name": "python3-libselinux",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6d6adc4ef5c889d481eebb9c69006c2736aaa5f714698ea0d0ab4faddc63b2fa&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libselinux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-3.el9"
        },
        "version": "3.6-3.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "16",
        "kind": "binary",
        "name": "quota-nls",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa4ea9ef89ca919ef069b1f4e8d2ae2111aa0a93e82a905917734f6fc22da0ca0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "quota",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.09-4.el9"
        },
        "version": "1:4.09-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "30",
        "kind": "binary",
        "name": "libzstd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aad52e1bd45f92f06d4d52c1c40c169f6c62cdae5f19bf99f3c6ba19b36e828c6&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "zstd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.5-1.el9"
        },
        "version": "1.5.5-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "142",
        "kind": "binary",
        "name": "libcbor",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aaf6eb3857cf6a4b485b627f9a8f228bcb5cc2e4ce01acc09a14ff45cbd969469&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libcbor",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.7.0-5.el9"
        },
        "version": "0.7.0-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "310",
        "kind": "binary",
        "name": "kexec-tools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1890848df1dcc790a7ff10e7c2682b83a8045cd2d35f7eb62070493f26945338&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kexec-tools",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.0.29-5.el9_6.2"
        },
        "version": "2.0.29-5.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "441",
        "kind": "binary",
        "name": "perl-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7c482cbb79aa5bb28f42ed634dd8f3ebab7ee6d399b8ccfe0074e43fd13aa5ac&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "4:5.32.1-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "48",
        "kind": "binary",
        "name": "alternatives",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af2acabdb1ba98d5997b31790e6d1dcf511cc35b9adf5f4e682e50a277e4a647e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "chkconfig",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.24-2.el9"
        },
        "version": "1.24-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "80",
        "kind": "binary",
        "name": "libverto",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9df6718fe2096ff3275b00917aa8cc21aff1b426cd151b6ad3a823c49331705f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libverto",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.3.2-3.el9"
        },
        "version": "0.3.2-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "246",
        "kind": "binary",
        "name": "logrotate",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A63f307839cf782b6c36ea3e2a36cd4f4e9f42ae8740e69807bc837e7ce2bb4cc&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "logrotate",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.18.0-9.el9"
        },
        "version": "3.18.0-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "75",
        "kind": "binary",
        "name": "iproute",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1195c3e232864774f061d1bfaf11ec9348793d705bd34d6da509f16fc1fb0051&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "iproute",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.11.0-1.el9"
        },
        "version": "6.11.0-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "111",
        "kind": "binary",
        "name": "mpfr",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abb2f0e3dd8e38f7d97da20424eb4eebed8f48a1edcff56236fa2910763b0100f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "mpfr",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.1.0-7.el9"
        },
        "version": "4.1.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "120",
        "kind": "binary",
        "name": "libibverbs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad291b087f10598aa2ebbbc00995408cf51a882f7a7c4d021ba694d25bc3d8878&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rdma-core",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "54.0-1.el9"
        },
        "version": "54.0-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "391",
        "kind": "binary",
        "name": "perl-libnet",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aab66d7912c4df2e809d6564726d84b7010d80e1bf264f2d0000945008bf8d5e5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-libnet",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.13-4.el9"
        },
        "version": "3.13-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "156",
        "kind": "binary",
        "name": "libtool-ltdl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A45a0d0e056c45a08d8b50ce8a666626cbe8f9c21debb6f6adfa92452e7d198d9&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libtool",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.4.6-46.el9"
        },
        "version": "2.4.6-46.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "328",
        "kind": "binary",
        "name": "python3-dateutil",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5a1f34424c4acbf3099143493de2bedf5be4b8e9b89e818926d5d9a2160dbdaa&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-dateutil",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.8.1-7.el9"
        },
        "version": "1:2.8.1-7.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "412",
        "kind": "binary",
        "name": "perl-HTTP-Tiny",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2c4276ddd9862bc1ea27800dcde65dd269024609b660d37b447661e99173d4a1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-HTTP-Tiny",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.076-462.el9"
        },
        "version": "0.076-462.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "448",
        "kind": "binary",
        "name": "cockpit-ws",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0b438334a09925af783c14e22c68a791daf20e5c2c0b88951c63a62c65cb06ee",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cockpit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "334.2-1.el9_6"
        },
        "version": "334.2-1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "373",
        "kind": "binary",
        "name": "python3-attrs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae23c1e4815651f4458a1e49894529f16df4dbc37b02848b72bf734ee6f83a132&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-attrs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20.3.0-7.el9"
        },
        "version": "20.3.0-7.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "402",
        "kind": "binary",
        "name": "perl-Text-Tabs+Wrap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A390fca7556405d4022ddc263ca4cbc479bba180a04f6334242893df86de5e4a1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Text-Tabs+Wrap",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2013.0523-460.el9"
        },
        "version": "2013.0523-460.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "420",
        "kind": "binary",
        "name": "perl-Text-ParseWords",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A83353098cd2b41936d3e146e15b433db5f4c6ab777004153ffbc0c70ca66c7e4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Text-ParseWords",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.30-460.el9"
        },
        "version": "3.30-460.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "435",
        "kind": "binary",
        "name": "perl-Getopt-Long",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0b88e101473ecc1889f521f0c0ebff17200fb0c02b69578a12307ac320d6b3e4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Getopt-Long",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.52-4.el9"
        },
        "version": "1:2.52-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "194",
        "kind": "binary",
        "name": "kmod",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ada23c8780b1b4d1fdd219a10458b5b39ddbd987f0e77e0ff5f37a50b37f2d2b6&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kmod",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "28-10.el9"
        },
        "version": "28-10.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "227",
        "kind": "binary",
        "name": "crontabs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9137e7c200c3ef3259f8a82edd3dd8edc804575f81288cca2e69c3dc75191e1c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "crontabs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.11-27.20190603git.el9_0"
        },
        "version": "1.11-27.20190603git.el9_0"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "290",
        "kind": "binary",
        "name": "dracut-network",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0c5652283ed0420628dd856bfbf6bcfce3fa89c4f0164e2b4fc158b611f94bde&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dracut",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "057-88.git20250311.el9_6"
        },
        "version": "057-88.git20250311.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "440",
        "kind": "binary",
        "name": "perl-Encode",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab83ab21107e14a15293973be0cff3d4ab196054a3fa709fb8b36ff722c378e27&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Encode",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.08-462.el9"
        },
        "version": "4:3.08-462.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "442",
        "kind": "binary",
        "name": "perl-interpreter",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A84d913c4dad8e35e9245422821a569ffe18c07482fe0cf10a9b7e65e2f4b9912&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "4:5.32.1-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "248",
        "kind": "binary",
        "name": "PackageKit-glib",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7812af402169d4d93837f6f336bdc44acec657e0a76cc20074bc9c7790d83dce&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "PackageKit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.6-1.el9"
        },
        "version": "1.2.6-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "362",
        "kind": "binary",
        "name": "python3-pysocks",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af4b88cb8b212f231aa5e04b5c51c2fc3ef05416742104b927047c59ab71b9f53&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-pysocks",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.7.1-12.el9"
        },
        "version": "1.7.1-12.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "485",
        "kind": "binary",
        "name": "libsysfs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad3073ffa222ce750cfd26a542dac4bd89c38719862b09bac1601e554c218441c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sysfsutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.1.1-10.el9"
        },
        "version": "2.1.1-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "101",
        "kind": "binary",
        "name": "libsigsegv",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adea2a137c8f1b440c5256d46e95c80314c60b6862d5c12eee0e9a34e4b8859e4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libsigsegv",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.13-4.el9"
        },
        "version": "2.13-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "168",
        "kind": "binary",
        "name": "libjpeg-turbo",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A150c6f41cd74a83df63fc465050dc977b3ff9dea715542661d413bce54f7afc3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libjpeg-turbo",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.0.90-7.el9"
        },
        "version": "2.0.90-7.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "344",
        "kind": "binary",
        "name": "python3-configobj",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa3e7a2891bc0347ddb7dd0ace29e921a9eb6976d50ce7cec5021c3c36ab0bc8a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-configobj",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.0.6-25.el9"
        },
        "version": "5.0.6-25.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "399",
        "kind": "binary",
        "name": "perl-IO-Socket-SSL",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4b161d2808fa928160005ea6b3af5dc7a760d0a2c8be6e3cce30206ddc6b8be3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-IO-Socket-SSL",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.073-2.el9"
        },
        "version": "2.073-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "506",
        "kind": "binary",
        "name": "shim-x64",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa47ec293302b18f51e6a2c5f1b84ecf7abd152146ccc45fce0e5e01e5c983a68&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "shim",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "15.8-4.el9_3"
        },
        "version": "15.8-4.el9_3"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "70",
        "kind": "binary",
        "name": "libbpf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A712dfe8e652da554449df288b84cd80d3ad6c0a9ea364b47041c6c64ad9ab79a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libbpf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.0-1.el9"
        },
        "version": "2:1.5.0-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "98",
        "kind": "binary",
        "name": "libpath_utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A497c30acc30edfbcf0fc59f3cad16eebb69996dbb961bb549a7a6975abf0f2ae&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ding-libs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.1-53.el9"
        },
        "version": "0.2.1-53.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "297",
        "kind": "binary",
        "name": "fwupd-plugin-flashrom",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1e84d023aed16ea763347f8ceed304e11e1f4c9d6e69faeeeb6bc4d4b260b3b2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "fwupd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.9.26-1.el9"
        },
        "version": "1.9.26-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "489",
        "kind": "binary",
        "name": "kernel-core",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abf82d33b74e9c872169b79eecd2a943332ff88591fda71fd729e931600d83a79&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms&repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "5.14.0-570.49.1.el9_6"
        },
        "version": "5.14.0-570.49.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "153",
        "kind": "binary",
        "name": "libpipeline",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af371bd6a3a0e7db2f8ad208084a1f7dcf87744ff4a75e9a726151b2ed6d1239a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libpipeline",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.3-4.el9"
        },
        "version": "1.5.3-4.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "414",
        "kind": "binary",
        "name": "perl-SelectSaver",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A948cd7de6ced9d8d0ef54f906f77906ecc58a7942cfbc99b63e940d06601296e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.02-481.1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "3",
        "kind": "binary",
        "name": "fonts-filesystem",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A452d302e2592174240aea2a3b27820bd64386c40856e8f61c1db90e9b4538d15&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "fonts-rpm-macros",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.0.5-7.el9.1"
        },
        "version": "1:2.0.5-7.el9.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "27",
        "kind": "binary",
        "name": "zlib",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab3ef882b84f6a5c77baba411cff1843dfb7ff869b25f81b9395d85135faa2ef0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "zlib",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.11-40.el9"
        },
        "version": "1.2.11-40.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "284",
        "kind": "binary",
        "name": "kernel-modules-core",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4e0a89ead6d795cc1a5992b44099cd66e536a6889f9436bb6f0fb285b555ef85",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.14.0-570.41.1.el9_6"
        },
        "version": "5.14.0-570.41.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "108",
        "kind": "binary",
        "name": "newt",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1283fa2ce96ab0a07f323b525a321a79ab0f9d6ca69020a1a321bf67992d204f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "newt",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.52.21-11.el9"
        },
        "version": "0.52.21-11.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "385",
        "kind": "binary",
        "name": "python3-jinja2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A30ef7e32662e59a75364c20838dea6468d5b762ff8cd41b7ddb887cf4b1588ab&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-jinja2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.11.3-8.el9_5"
        },
        "version": "2.11.3-8.el9_5"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "12",
        "kind": "binary",
        "name": "setup",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3c752e15a42e98ca2494187fddf6c3a59d438200e7a00a871a22d26c9a197fa3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "setup",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.13.7-10.el9"
        },
        "version": "2.13.7-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "45",
        "kind": "binary",
        "name": "libtalloc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6a623e272b33832a046a08c140109fb108468ad892faac550409ba9d8659a47f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libtalloc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.4.2-1.el9"
        },
        "version": "2.4.2-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "115",
        "kind": "binary",
        "name": "tar",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A62768e2b5609e1c53e30022609e088850769a929d5da74a48f0ebd23bc22cbc1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "tar",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.34-7.el9"
        },
        "version": "2:1.34-7.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "353",
        "kind": "binary",
        "name": "python3-dnf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A454abb94a5847ed875044ede6155b83ce38d9e454278ed036e09ab196658c21f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dnf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.14.0-25.el9"
        },
        "version": "4.14.0-25.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "493",
        "kind": "binary",
        "name": "p11-kit-trust",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A22c83ab22219e448f3eed3aac1ba022e6a6237ead5cd1a8b8c86487e2bc16a1d&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "p11-kit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "0.25.3-3.el9_5"
        },
        "version": "0.25.3-3.el9_5"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "20",
        "kind": "binary",
        "name": "ncurses-base",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abe77ec298361d33952fc20b4790aae408792009692c9ae64ceb927e7297e117a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ncurses",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.2-10.20210508.el9_6.2"
        },
        "version": "6.2-10.20210508.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "55",
        "kind": "binary",
        "name": "lz4-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af7a7ca5de82c03a63a7db2d16e0fbdad2c9b291ef0c65b929536dc15b4d0a0a4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lz4",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.9.3-5.el9"
        },
        "version": "1.9.3-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "88",
        "kind": "binary",
        "name": "e2fsprogs-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa73b9e580b54235ec99be44114cfce149cca0c33cbcaf03d6a8263c783934070&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "e2fsprogs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.46.5-7.el9"
        },
        "version": "1.46.5-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "325",
        "kind": "binary",
        "name": "python3-gobject-base",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6ff628e3e01c35f06aca10c0399fa44614951411494203181afdcd42701d09b2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pygobject3",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.40.1-6.el9"
        },
        "version": "3.40.1-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "421",
        "kind": "binary",
        "name": "perl-mro",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac996bcdd6778b3c1b7dd0bd09a3327bd9e23816492afe7a83e3842fa2c43529f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.23-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "169",
        "kind": "binary",
        "name": "libmaxminddb",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ade28fa9c116bb93323c5657e72356bff9e086ae278cc34db579f5d1ceeb05f0f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libmaxminddb",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.2-4.el9"
        },
        "version": "1.5.2-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "47",
        "kind": "binary",
        "name": "lua-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3775140f398725124015ec45890e35f8ae5ce50615e511f9686140e54e00b716&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lua",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.4.4-4.el9"
        },
        "version": "5.4.4-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "193",
        "kind": "binary",
        "name": "json-glib",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4bed13c3b19845a4b45f1d96224b25eabcc2ca171d11b0743b5da8e84f6222ec&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "json-glib",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.6.6-1.el9"
        },
        "version": "1.6.6-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "254",
        "kind": "binary",
        "name": "gdk-pixbuf2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A16867643c40e7397bd1470959224eae3acc9a6f51a5f1d2af476c2c4d7b9d929&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gdk-pixbuf2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.42.6-6.el9_6"
        },
        "version": "2.42.6-6.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "336",
        "kind": "binary",
        "name": "python3-idna",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa6b1cebecb10a01589e26df13b8873cc4dffa872281f22ee4480763e00690159&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-idna",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.10-7.el9_4.1"
        },
        "version": "2.10-7.el9_4.1"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "346",
        "kind": "binary",
        "name": "python3-policycoreutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7433070dced6c0e438d31a942254eb63b8633683cd394ba41aecf900b2d56441&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "policycoreutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-2.1.el9"
        },
        "version": "3.6-2.1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "363",
        "kind": "binary",
        "name": "python3-urllib3",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af467f6b0fa93aad0e4eb80a84307135a03f1433dfec95b20837a261d847a5eb5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-urllib3",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.26.5-6.el9"
        },
        "version": "1.26.5-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "482",
        "kind": "binary",
        "name": "lshw",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3e3db4d1741f296c43014a887b37e52e798e5cde1cb70e908a467fc6bc40002c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lshw",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "B.02.20-1.el9"
        },
        "version": "B.02.20-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "208",
        "kind": "binary",
        "name": "procps-ng",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A00c5f3a236303bb3dd58efce53b5f5cb759c1761f2393744410ffbe3080b1b36&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "procps-ng",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.3.17-14.el9"
        },
        "version": "3.3.17-14.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "79",
        "kind": "binary",
        "name": "libtdb",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A407163d37184fbb30400a5a7081fb676747bba9f63191c76f873940a80d8f720&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libtdb",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.4.12-1.el9"
        },
        "version": "1.4.12-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "125",
        "kind": "binary",
        "name": "libksba",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae57fed9feceaa09759a6d02087ade587841d8f46f082b938b44376e1baf94922&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libksba",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.1-7.el9"
        },
        "version": "1.5.1-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "263",
        "kind": "binary",
        "name": "oddjob-mkhomedir",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aad3a25666cf6ab7287531ef02ecbbecdc82f03e33595558733bdb2a60f541b10&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "oddjob",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.34.7-7.el9"
        },
        "version": "0.34.7-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "266",
        "kind": "binary",
        "name": "cyrus-sasl-lib",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6bf8a8e1fb6eddcda76b15db9708cdb44967210e46f860a4b9d249a5f2bea29a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cyrus-sasl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.1.27-21.el9"
        },
        "version": "2.1.27-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "148",
        "kind": "binary",
        "name": "libndp",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A165a0e15c85a954224e2ea980d1f34c2350d01e2239d3b04e472e5e3d49fe2c3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libndp",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.9-1.el9"
        },
        "version": "1.9-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "150",
        "kind": "binary",
        "name": "libnetfilter_conntrack",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A76de7927ed58ca15619e733e930773da34806731c9ea659cb0a3eb3ef860cdf2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libnetfilter_conntrack",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.9-1.el9"
        },
        "version": "1.0.9-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "152",
        "kind": "binary",
        "name": "libnghttp2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A82b9a184166b675ef553fcab14e36c7fde86912397572924646775e3d86ad8b6&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "nghttp2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.43.0-6.el9"
        },
        "version": "1.43.0-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "159",
        "kind": "binary",
        "name": "nettle",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0eea9b0d5e09190d63d40c8fba3f6e64ed7455a41469a02adc78e99d6f9ab0bb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "nettle",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.10.1-1.el9"
        },
        "version": "3.10.1-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "277",
        "kind": "binary",
        "name": "libssh",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A60fc4d9ed8fce4e0d058c6946e88386ccdebc62229c4034224c9644ca55e7ce8&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libssh",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.10.4-13.el9"
        },
        "version": "0.10.4-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "157",
        "kind": "binary",
        "name": "libtraceevent",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2ada470420e2181c29bbcca81dc82d5861af1cc4c5a88accbe7d66e8cfb6e56d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libtraceevent",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.3-3.el9"
        },
        "version": "1.5.3-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "190",
        "kind": "binary",
        "name": "dbus-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae7a442c6c4cba130a725a91b4d6cf37ec6e82241d55d19b38a58ede829f7d93a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dbus",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.12.20-8.el9"
        },
        "version": "1:1.12.20-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "205",
        "kind": "binary",
        "name": "NetworkManager-libnm",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A98db3eb27d8ed1e44448f6331b9615e5a4605bc3986c9415a72393e8a6b0445b",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "NetworkManager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.52.0-7.el9_6"
        },
        "version": "1:1.52.0-7.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "316",
        "kind": "binary",
        "name": "rsyslog",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A91d24f290a07d7b352a28f1434ee5503eba51fe71a29b4120aeddf7e453384b7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rsyslog",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.2412.0-1.el9"
        },
        "version": "8.2412.0-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "436",
        "kind": "binary",
        "name": "perl-Carp",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa6c0c1e765da10c86daf9ac3e1ede310c5abba40c68f8acc6aa9a5dd4da621cf&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Carp",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.50-460.el9"
        },
        "version": "1.50-460.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "11",
        "kind": "binary",
        "name": "redhat-release",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A95a2803525ce2956bbff45b1fd20358aae81b3aa3b29ca3ef45f2100c8b04253&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "redhat-release",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "9.6-0.1.el9"
        },
        "version": "9.6-0.1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "375",
        "kind": "binary",
        "name": "python3-jsonpatch",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4110d238971efef1c5581b0646c8dc4867a303a0dbce7e06347b6a4091aced3c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-jsonpatch",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.21-16.el9"
        },
        "version": "1.21-16.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "41",
        "kind": "binary",
        "name": "crypto-policies",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A31b9be21c5316e59738cfed16b78bc0b98471c2b49b16881d8be392de540cb5a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "crypto-policies",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20250128-1.git5269e22.el9"
        },
        "version": "20250128-1.git5269e22.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "76",
        "kind": "binary",
        "name": "gmp",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A10a99865037f7043b6b4de265b48d4e1e08634a675deb58553043b784bad917e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gmp",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.2.0-13.el9"
        },
        "version": "1:6.2.0-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "151",
        "kind": "binary",
        "name": "iptables-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5978b2c0a7513ae79f0fec3ad0677a981dbacd3a13b02b6b8a16f6ae13737017&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "iptables",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.8.10-11.el9_5"
        },
        "version": "1.8.10-11.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "215",
        "kind": "binary",
        "name": "dbus",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aaaada29e7a9cab643ee3cc4eee876ea240668b776a129e9787b275f57c1e91d5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dbus",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.12.20-8.el9"
        },
        "version": "1:1.12.20-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "245",
        "kind": "binary",
        "name": "libkcapi-hmaccalc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0fdd2b552c2f4a097006f8684e710001a89ba1f60fca5a6964e33f476ab8511d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libkcapi",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.4.0-2.el9"
        },
        "version": "1.4.0-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "293",
        "kind": "binary",
        "name": "os-prober",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac7cc8e75adec8afad4a26afc674b5d7d1fd370da3299c2ee51e47f220f691083&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "os-prober",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.77-12.el9_5"
        },
        "version": "1.77-12.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "334",
        "kind": "binary",
        "name": "python3-hawkey",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A31830a5a6bcba366bd58c5466ed2ed4865285e1c4f3e33284f0f26f761122e09&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libdnf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.69.0-13.el9"
        },
        "version": "0.69.0-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "265",
        "kind": "binary",
        "name": "e2fsprogs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae26852cba6af922219b46cb59dea44bafceb116e3e91d43c4fa40d5db2c47f9d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "e2fsprogs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.46.5-7.el9"
        },
        "version": "1.46.5-7.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "318",
        "kind": "binary",
        "name": "python-unversioned-command",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac0a52e1d927aabd1d98dd67f93de705c541b44b0d957884dc8c2616d0932cc67",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python3.9",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.9.21-2.el9_6.2"
        },
        "version": "3.9.21-2.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "320",
        "kind": "binary",
        "name": "python3-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7c8dac1dc8bf5ed6326506fe425f83ac92ce7fcf840e699fa8f06ec6a10970dc",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python3.9",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.9.21-2.el9_6.2"
        },
        "version": "3.9.21-2.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "341",
        "kind": "binary",
        "name": "python3-dasbus",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A238967f6219411c7464805ebe2315780df53c2da05e5780b038cb3d6ddf0e259&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-dasbus",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.4-5.el9"
        },
        "version": "1.4-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "349",
        "kind": "binary",
        "name": "python3-librepo",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0b21b889b3c8753f4f666346a86472743f0be26dd498a181aecc4a530b95911a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "librepo",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.14.5-2.el9"
        },
        "version": "1.14.5-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "469",
        "kind": "binary",
        "name": "chrony",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6846620b197f4677456c61b763c8cf54e74dd5d6243f213e5eea90afed35832e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "chrony",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.6.1-1.el9"
        },
        "version": "4.6.1-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "226",
        "kind": "binary",
        "name": "cronie",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5b8fa9bfc2aabd425365bb3d894b584114f933df8746046eb9716a647fa66c82&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cronie",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.7-14.el9_6"
        },
        "version": "1.5.7-14.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "357",
        "kind": "binary",
        "name": "python3-libxml2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9f86be7ed7715cca7fc1017eaa5b1ab74898ed143f6778e772ffd8c6cb151bf0",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libxml2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.13-12.el9_6"
        },
        "version": "2.9.13-12.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "29",
        "kind": "binary",
        "name": "popt",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Afa45b1f1cd791b775cb4638d2ef3297a01e964969dfafec3c95d72d1a264b6a1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "popt",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.18-8.el9"
        },
        "version": "1.18-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "95",
        "kind": "binary",
        "name": "libbrotli",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4430723e3eab6bcaafdcaa0260a53cd17cc6f2065c48db8644b862f119cb6333&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "brotli",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.9-7.el9_5"
        },
        "version": "1.0.9-7.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "204",
        "kind": "binary",
        "name": "cracklib",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A33f5fd720f4828c90c930fb3129964a9fec5aabb8fd3d4620b6d2aaa22d67910&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cracklib",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-27.el9"
        },
        "version": "2.9.6-27.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "264",
        "kind": "binary",
        "name": "authselect",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A656a2efeebc5abe0b9015bee734bfce736027fff73edbc20c16f46942c646787&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "authselect",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.6-3.el9"
        },
        "version": "1.2.6-3.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "407",
        "kind": "binary",
        "name": "perl-IPC-Open3",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2bee61b51e624e022dd07b2104c0f9333be7d98854d4672060fc42791a3177ec&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.21-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "460",
        "kind": "binary",
        "name": "rpm-plugin-audit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3fab3a86800254013d92cb330d2144e0c94b286fdaf21515b0946351268985ae&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "497",
        "kind": "binary",
        "name": "kernel",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae04ca62b73a902f5546e565600bbc0027e7b011fb79d210f0f27f741b93a2cc9&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms&repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "5.14.0-570.49.1.el9_6"
        },
        "version": "5.14.0-570.49.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "129",
        "kind": "binary",
        "name": "libicu",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7b3d959c2da798f58a3b1a01cee5b927ea9506ccba0bd8546f2898070e30cc1e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "icu",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "67.1-10.el9_6"
        },
        "version": "67.1-10.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "418",
        "kind": "binary",
        "name": "perl-Pod-Perldoc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A35c250e76eb0fb1878bd74755eb0697c7f953a45266f2fb1dad1c75b51837d67&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Pod-Perldoc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.28.01-461.el9"
        },
        "version": "3.28.01-461.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "456",
        "kind": "binary",
        "name": "nfs-utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Acc37a4dd257f5ec7819fa136315804b33539360dd55a8236ac9f884c4ce75fdb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "nfs-utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5.4-34.el9"
        },
        "version": "1:2.5.4-34.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "99",
        "kind": "binary",
        "name": "libini_config",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A541a604fd1f63d9233ce910df7f1c4094b65a42c1f734b0a21b9d23dd1cca742&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ding-libs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.1-53.el9"
        },
        "version": "1.3.1-53.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "117",
        "kind": "binary",
        "name": "gettext-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A470a7dee1b96db200e3f32ba3892c2b3232b05a0546a01713f444d0aa76b4225&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gettext",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.21-8.el9"
        },
        "version": "0.21-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "143",
        "kind": "binary",
        "name": "libdaemon",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4d3585a2a8fb32096ee4672ac58bdb17feaaca32c461009835092453e81a66f0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libdaemon",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.14-23.el9"
        },
        "version": "0.14-23.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "145",
        "kind": "binary",
        "name": "libeconf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1891e1429ee43ed33a8ba647245959b8e6dd3816f51742517a99428b48128cfb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libeconf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.4.1-4.el9"
        },
        "version": "0.4.1-4.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "332",
        "kind": "binary",
        "name": "python3-distro",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0b670ef346b89814a205f580ea723197c69461cbd0f832cc79edb7151dc38670&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-distro",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.0-7.el9"
        },
        "version": "1.5.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "184",
        "kind": "binary",
        "name": "openssl-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae31fab80c51fbb6eb7aed86445e3c739ae11d4041b496e795818587c27aea5d0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openssl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.2.2-6.el9_5.1"
        },
        "version": "1:3.2.2-6.el9_5.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "198",
        "kind": "binary",
        "name": "util-linux-core",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab4eee088bb7932ce614f81e267b4efe4c138c6056cd3779909882ba833a51e8c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "util-linux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.37.4-21.el9"
        },
        "version": "2.37.4-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "468",
        "kind": "binary",
        "name": "openssh-server",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6ec885df036233a31be3f7d963e050825076072fc8e6c715c6695633c7434eba&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openssh",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.7p1-45.el9"
        },
        "version": "8.7p1-45.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "77",
        "kind": "binary",
        "name": "libref_array",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae88e0e4301552155eee6c97d5dd299ded8ea2e15233c2a9dcfd02867fcf6a105&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ding-libs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.1-53.el9"
        },
        "version": "0.1.5-53.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "181",
        "kind": "binary",
        "name": "coreutils-common",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3beeadda4e37bb9f4e8aaba710a80b29e81881d282b82e79936f9258d4e91dca&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "coreutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.32-39.el9"
        },
        "version": "8.32-39.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "463",
        "kind": "binary",
        "name": "openssh-clients",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abd62970bd4b1e0027dcc2ea47f356861c3cd0a78017e04b35cb28d23c7ee2f77&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openssh",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.7p1-45.el9"
        },
        "version": "8.7p1-45.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "54",
        "kind": "binary",
        "name": "libsepol",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A28e0e88f38016ee9d2723af664887a9c51a01bf888a7744aabce85f2378bc642&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libsepol",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-2.el9"
        },
        "version": "3.6-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "69",
        "kind": "binary",
        "name": "libgcrypt",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af32f2f2282c6afa91afbbf8e8677e7c69548485a73093607da2f1df05ea7b578&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libgcrypt",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.10.0-11.el9"
        },
        "version": "1.10.0-11.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "398",
        "kind": "binary",
        "name": "perl-File-Path",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa0dff71e7696327269120b5bac5a0b4005395f1aff3394787b11a603f084f4f2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-File-Path",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.18-4.el9"
        },
        "version": "2.18-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "104",
        "kind": "binary",
        "name": "lzo",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A84be856d9ef9d1a5319d8e41624bcad2951950b96740f108f5a422c101fd08f1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lzo",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.10-7.el9"
        },
        "version": "2.10-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "90",
        "kind": "binary",
        "name": "libproxy",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A724d76bf600fc66048f3907d4852605d58c7c88bf73aa3b4c46fdd28e8c0ad7c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libproxy",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.4.15-35.el9"
        },
        "version": "0.4.15-35.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "317",
        "kind": "binary",
        "name": "python3-pip-wheel",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3f4a2d28d24e53b698a51b5519a9a7aced257c27123b887c3ee15e1d48e56c3e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-pip",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "21.3.1-1.el9"
        },
        "version": "21.3.1-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "369",
        "kind": "binary",
        "name": "teamd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9223039716f48aad3fe6b31d6f9c614b0b56297caf5b27d11fc3c4954332f38e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libteam",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.31-16.el9_1"
        },
        "version": "1.31-16.el9_1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "471",
        "kind": "binary",
        "name": "qemu-guest-agent",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac68d49fb7df36b418452a033f6613e463072cf353441cf378d4a2fe061192fb0",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "qemu-kvm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "9.1.0-15.el9_6.9"
        },
        "version": "17:9.1.0-15.el9_6.9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "103",
        "kind": "binary",
        "name": "libyaml",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A384ea82813a6ebecc6a2a0fe47db5be534fb4d1dc2f4655f4d9e64d0549ba52c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libyaml",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.2.5-7.el9"
        },
        "version": "0.2.5-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "203",
        "kind": "binary",
        "name": "libarchive",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae9c44108ba2924eb34bbd710de9a5d2dfd52e9e083a029c6aa589de9b2fb67e5",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libarchive",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.5.3-6.el9_6"
        },
        "version": "3.5.3-6.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "206",
        "kind": "binary",
        "name": "gobject-introspection",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4efe67d13ce247098b228fd9d5e7d13c5511ba119d3037c1d44e302e3977c99f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gobject-introspection",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.68.0-11.el9"
        },
        "version": "1.68.0-11.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "276",
        "kind": "binary",
        "name": "sudo",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af33b9a00295fec4db582dbbc17447d381ad28bace68181d6b81169a3c4cf3361",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sudo",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.9.5p2-10.el9_6.2"
        },
        "version": "1.9.5p2-10.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "322",
        "kind": "binary",
        "name": "python3-six",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A793034cfeaacbc2f77f349dbaa42d1b8609827eabd0dcfcd93d43d53eb9c364b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-six",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.15.0-9.el9"
        },
        "version": "1.15.0-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "428",
        "kind": "binary",
        "name": "perl-MIME-Base64",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4746b06f0106dc09d85c3cc2cbee1a3f156b9a461e229204a41726994f8abd4d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-MIME-Base64",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.16-4.el9"
        },
        "version": "3.16-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "23",
        "kind": "binary",
        "name": "glibc-common",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad8cc1ffa3592dff11055da93fd6b2ccf43280588dc9e9ba1b59e806c1305745e",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "glibc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.34-168.el9_6.23"
        },
        "version": "2.34-168.el9_6.23"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "86",
        "kind": "binary",
        "name": "libassuan",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1be2e3c5d1b68883bfd4a973518042cb88a4181063fc3f8e081bf5495f56a89f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libassuan",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5.5-3.el9"
        },
        "version": "2.5.5-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "160",
        "kind": "binary",
        "name": "npth",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7a7ba1657f8c46e48416a83de438f2d33ad6609676b458d11809624a3ef0029e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "npth",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.6-8.el9"
        },
        "version": "1.6-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "163",
        "kind": "binary",
        "name": "sg3_utils-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab19df97ee1e84aeb06a55076a2911eb85a5cb526d4dc3492630ded5e757c9c91&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sg3_utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.47-10.el9"
        },
        "version": "1.47-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "256",
        "kind": "binary",
        "name": "gsettings-desktop-schemas",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aac2e2cf121e8640a1804377f3514307b8b7ff2fb3e04885445ee5a095d9d9f8e",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gsettings-desktop-schemas",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "40.0-7.el9_6"
        },
        "version": "40.0-7.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "392",
        "kind": "binary",
        "name": "perl-AutoLoader",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A72de989f41cce0deb218547e66b9d2a9cadc98eb51cb551f438e80b0d83d626a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "5.74-481.1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "9",
        "kind": "binary",
        "name": "rhsm-icons",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A822761c6824bb6c7c79b6870363722552369d1caf53a27b7282ed1917ff7190c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "subscription-manager-cockpit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6-2.el9"
        },
        "version": "6-2.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "409",
        "kind": "binary",
        "name": "perl-File-Temp",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0634d0bfdf0eeda4c53fd44591a96584d7b9c5f5f512c7586102653ca50bc2cf&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-File-Temp",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.231.100-4.el9"
        },
        "version": "1:0.231.100-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "464",
        "kind": "binary",
        "name": "kernel",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae04ca62b73a902f5546e565600bbc0027e7b011fb79d210f0f27f741b93a2cc9",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.14.0-570.41.1.el9_6"
        },
        "version": "5.14.0-570.41.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "147",
        "kind": "binary",
        "name": "libverto-libev",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3ec4edb131d031fdc9fc838e8c5b5de59b8c0f5995bb385672cf0845d65ba7f1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libverto",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.3.2-3.el9"
        },
        "version": "0.3.2-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "387",
        "kind": "binary",
        "name": "perl-Digest-MD5",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3bbfee9260dcafc081608acaecd747bce73d700a2ad01fdc465d6ef7d512e8bb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Digest-MD5",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.58-4.el9"
        },
        "version": "2.58-4.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "2",
        "kind": "binary",
        "name": "hwdata",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4c5cc65fb49baf8ae17248609eaf3fe0e88c70cd2e90a83eeabb0a4847fd0cef&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "hwdata",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.348-9.18.el9"
        },
        "version": "0.348-9.18.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "8",
        "kind": "binary",
        "name": "gawk-all-langpacks",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A69fc1ab20ffc8c0caa6b893e78b30aa040be1d023e7ccd12d695bade27553aa4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gawk",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.1.0-6.el9"
        },
        "version": "5.1.0-6.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "450",
        "kind": "binary",
        "name": "tuned",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A72d8e48a1e2d8f52d07a343c5b26d1a61f71a0f03ed2e03f16b9281173b5bfed&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "tuned",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.25.1-2.el9_6"
        },
        "version": "2.25.1-2.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "197",
        "kind": "binary",
        "name": "kmod-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A26c6481cb4c11f929af4596c3a1814f9f4da87db2f0e9e224f8fc7af87c7c1c2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kmod",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "28-10.el9"
        },
        "version": "28-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "223",
        "kind": "binary",
        "name": "polkit-pkla-compat",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A766fa2dbd8156310325c0d3289b4dcc7a27c3b812f3b374e3a3fa5bc69288e25&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "polkit-pkla-compat",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.1-21.el9"
        },
        "version": "0.1-21.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "364",
        "kind": "binary",
        "name": "python3-requests",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A63fac8a8b64792e7c0ce14cdf311add426fb36576cdc75d59dc429c468120b4e",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-requests",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.25.1-10.el9_6"
        },
        "version": "2.25.1-10.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "484",
        "kind": "binary",
        "name": "dosfstools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9e4061060388817f678efa07effc588e8f3c947eb1d58063c286340378ba8b18&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dosfstools",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.2-3.el9"
        },
        "version": "4.2-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "162",
        "kind": "binary",
        "name": "jq",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa7f67ae12ffd15dc9b175f3bcb2db144e2daf7d6e3491d5ab2dfcc88c3757713&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "jq",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.6-17.el9_6.2"
        },
        "version": "1.6-17.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "311",
        "kind": "binary",
        "name": "rpcbind",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aef7346d6c04c12529a8aa0377d0ed5edebf56f6f08313524a9595e5809fff3ed&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpcbind",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.6-7.el9"
        },
        "version": "1.2.6-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "35",
        "kind": "binary",
        "name": "bzip2-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aee174790724e530baae49722b19b866c1e58f229ecfad78ad95be2a3a6c3786e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "bzip2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.8-10.el9_5"
        },
        "version": "1.0.8-10.el9_5"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "396",
        "kind": "binary",
        "name": "perl-IO-Socket-IP",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac4ccb0df0a93a29c289720484f04d561cd14fcf8bfa34a7edfeb54dbbe1ad644&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-IO-Socket-IP",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.41-5.el9"
        },
        "version": "0.41-5.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "408",
        "kind": "binary",
        "name": "perl-subs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A78c9ea8ac33a010b271e7b34d40d7fc22fd0d298eedc93240d721e52846f5823&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.03-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "466",
        "kind": "binary",
        "name": "dracut-config-generic",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A706609759dcd2db4dfb7688881fb6bde1a01905d54f4163c17757fdba966f242&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dracut",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "057-88.git20250311.el9_6"
        },
        "version": "057-88.git20250311.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "124",
        "kind": "binary",
        "name": "libcomps",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1bd7e1dade12ca4827714df7b691f3059919eac662fddb9d8ff7a60a24042c4f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libcomps",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.1.18-1.el9"
        },
        "version": "0.1.18-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "187",
        "kind": "binary",
        "name": "systemd-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aec827e973306a802efbd475888d2e30c8844df79d840f2444e5df0e8edd25ac1",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "systemd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "252-51.el9_6.2"
        },
        "version": "252-51.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "275",
        "kind": "binary",
        "name": "libjcat",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae5067874cb08a5b4c1d2fb174f9bff5ee29ca34a6afb0bc43ccc4d7c34e1f3ed&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libjcat",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.1.6-3.el9"
        },
        "version": "0.1.6-3.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "378",
        "kind": "binary",
        "name": "python3-oauthlib",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6403156469a64b75419c048d30ea898ee5e8a0b419d18583ca444c78efad44a0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-oauthlib",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.1.1-5.el9"
        },
        "version": "3.1.1-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "504",
        "kind": "binary",
        "name": "libgomp",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A275d138b355c250e77deeab028aa349cb176789fa650807590a782f08783e78f&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gcc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "11.5.0-5.el9_5"
        },
        "version": "11.5.0-5.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "73",
        "kind": "binary",
        "name": "libedit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa8b1518c45f924fa32bd08f84ce04093772dd9b1957ef056cde67919004f0150&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libedit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.1-38.20210216cvs.el9"
        },
        "version": "3.1-38.20210216cvs.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "395",
        "kind": "binary",
        "name": "perl-if",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aaa77df4a87fa164cd8fe6c6dab416725cdc373225321127bddceefcd9fa388ac&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "0.60.800-481.1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "324",
        "kind": "binary",
        "name": "python3-gobject-base-noarch",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A49881003cf6b81f3144723ed6eb2324a3892788601b6fb4c6f24c65abe339ca9&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pygobject3",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.40.1-6.el9"
        },
        "version": "3.40.1-6.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "343",
        "kind": "binary",
        "name": "python3-pyudev",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A394882f8e1e6b3607ff9c30f0ade7809c7a242e1e8574b50eb2720e960a7ea54&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-pyudev",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.22.0-6.el9"
        },
        "version": "0.22.0-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "461",
        "kind": "binary",
        "name": "grub2-efi-x64",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ada029aa77ca520cb4f6c71d0deb84e96380300704314020c66d494f88116ce6a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grub2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.06-104.el9_6"
        },
        "version": "1:2.06-104.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "64",
        "kind": "binary",
        "name": "sed",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A40ad87eee974a2f1e3a4ac38333e8f2302013d1d49ce82b2023046eee145c644&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sed",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.8-9.el9"
        },
        "version": "4.8-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "114",
        "kind": "binary",
        "name": "libselinux-utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9c889210b7c04aefb44daa38f807078b5d1e7079e9298f39cd1ce5a711614a10&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libselinux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-3.el9"
        },
        "version": "3.6-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "165",
        "kind": "binary",
        "name": "checkpolicy",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A25a0f8686a02c8334c308033bbb102484e9a519dc6da6973a8f4c0b94e4d5d0f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "checkpolicy",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-1.el9"
        },
        "version": "3.6-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "231",
        "kind": "binary",
        "name": "audit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9dd5c12e73cf1c7bd311b3f1cc369856e61b13987a73e7131d5d75fb18f34ca1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "audit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.1.5-4.el9"
        },
        "version": "3.1.5-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "329",
        "kind": "binary",
        "name": "python3-rpm",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac4749c74e0f60ade9e2bc60f3ff6804d06e0e94a2c73ad86ec06f24c1aeafcae&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "17",
        "kind": "binary",
        "name": "python3-setuptools-wheel",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A614eaeb0a5d69589d37a30e4f7c8cbed84a523a89d2d21cea9f69059f0d59fd1&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-setuptools",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "53.0.0-13.el9_6.1"
        },
        "version": "53.0.0-13.el9_6.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "253",
        "kind": "binary",
        "name": "libxmlb",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab37a7db62b720c9dab31bfca5ccb603a3e729bb6bde445503700489c3032c938&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libxmlb",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.3.10-1.el9"
        },
        "version": "0.3.10-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "66",
        "kind": "binary",
        "name": "shadow-utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A31daf5cef1e112808c425a3942545fc46c983dea535c3036c758c44dd9dde244&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "shadow-utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.9-12.el9"
        },
        "version": "2:4.9-12.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "179",
        "kind": "binary",
        "name": "kbd-legacy",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A00febc82c86111f67130a2914d5d3314a9d5cafeccd06c29b34d5769d7d6803d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kbd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.4.0-11.el9"
        },
        "version": "2.4.0-11.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "176",
        "kind": "binary",
        "name": "libreport-filesystem",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9a90b7eb74f47e6825720a54c5d1ad90bf31baadb6b42667710f55391d8b8b41&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libreport",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.15.2-6.el9"
        },
        "version": "2.15.2-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "110",
        "kind": "binary",
        "name": "squashfs-tools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac1e64a8e411bc33d5f3634eca2395359f6149026afc97074f54d472a0080b6dc&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "squashfs-tools",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.4-10.git1.el9"
        },
        "version": "4.4-10.git1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "19",
        "kind": "binary",
        "name": "pcre2-syntax",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5625ba68eb852651b82122e2d06d52b60348b1c8fae84e84c1e861078dc269a2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pcre2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "10.40-6.el9"
        },
        "version": "10.40-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "130",
        "kind": "binary",
        "name": "snappy",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A06dc13cd9032e8b70ca74360d18b817534825c859afd12544e4b71f45f4e6f74&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "snappy",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.1.8-8.el9"
        },
        "version": "1.1.8-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "298",
        "kind": "binary",
        "name": "ima-evm-utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa95150ae6d4ff02a4dd6c350b070316c6808d542dfc2db89f4f96f23d0b15454&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ima-evm-utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5-3.el9"
        },
        "version": "1.5-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "22",
        "kind": "binary",
        "name": "glibc-minimal-langpack",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa40c7d44c2842691af995cb8481a68716dd35b228919357e7ef5749f0104905d",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "glibc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.34-168.el9_6.23"
        },
        "version": "2.34-168.el9_6.23"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "133",
        "kind": "binary",
        "name": "hostname",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad0ef4fe83825ae2e52f1c139d2a09ac495cf4ce506eec65b8312c697019463fe&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "hostname",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.23-6.el9"
        },
        "version": "3.23-6.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "175",
        "kind": "binary",
        "name": "libssh-config",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab4b4001cf534ac33a4f039aff6c621719a4532ee4781f28522dd82108d115497&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libssh",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.10.4-13.el9"
        },
        "version": "0.10.4-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "235",
        "kind": "binary",
        "name": "device-mapper",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ada1f5900405dca9886fcca1d6e5d84f690c45d2c23f6fa856b8c5921821cce6b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lvm2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.03.28-6.el9"
        },
        "version": "9:1.02.202-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "413",
        "kind": "binary",
        "name": "perl-Socket",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8555d5ec2bf14a4e0fb42d0421a88263c1a3d14a4e698f5274a0373ff98a66e0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Socket",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.031-4.el9"
        },
        "version": "4:2.031-4.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "426",
        "kind": "binary",
        "name": "perl-File-Basename",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac3faa51c809fca2a88b5c5482cc5428b5754837de9d768087b3fb0c9b97d2a85&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "2.85-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "132",
        "kind": "binary",
        "name": "pigz",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A371fd62734fd750f2c93d00eccfc8ba55a9744a0a396036caf8c167aff17cdd7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pigz",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.8-1.el9"
        },
        "version": "2.8-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "192",
        "kind": "binary",
        "name": "glib2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa8cd583754427f07c4c74374012443b31530d9485c21629f87bb9ae2a1101319&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "glib2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.68.4-16.el9_6.2"
        },
        "version": "2.68.4-16.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "389",
        "kind": "binary",
        "name": "perl-FileHandle",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aaccb4d200932170291b916db7cff79362fa4c71e359c4ac07464108d96c345c3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "2.03-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "63",
        "kind": "binary",
        "name": "libselinux",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8e41179421107b1b6ab957874aac1e1983a62fa0c807bed87b524c8be6a3ce6e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libselinux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-3.el9"
        },
        "version": "3.6-3.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "173",
        "kind": "binary",
        "name": "linux-firmware-whence",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac4fcbba10d89034d786ec75046735b5cc18f83a880b66c0ad6857d2daba32ae0",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "linux-firmware",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20250812-151.4.el9_6"
        },
        "version": "20250812-151.4.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "295",
        "kind": "binary",
        "name": "flashrom",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A32f677c2bdf1eb465be5e9e5f8698ea0c2b8d535ccbe8e9eafa89628d0f75d0c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "flashrom",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2-10.el9"
        },
        "version": "1.2-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "443",
        "kind": "binary",
        "name": "redhat-logos",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aae8dda0749c3889ec3428ccbf6c5655ecc202e3364005cbfb6fa5acff2552981&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "redhat-logos",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "90.5-1.el9_6.1"
        },
        "version": "90.5-1.el9_6.1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "158",
        "kind": "binary",
        "name": "lmdb-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A62416b408bf2f059e3425489f5bccd40d7ddaa56f7f0e9624cb5f8c41b88217a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "lmdb",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.9.29-3.el9"
        },
        "version": "0.9.29-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "195",
        "kind": "binary",
        "name": "polkit-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac61c03e898a4873723cdca9c8af00b5bcaad84866de8b24abc81b766956a8f06&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "polkit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.117-13.el9"
        },
        "version": "0.117-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "212",
        "kind": "binary",
        "name": "libpwquality",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A83b2ff9694ec1831a8c8c9c8a5cf0aee622bb441716cd038d70e46df98d96b8e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libpwquality",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.4.4-8.el9"
        },
        "version": "1.4.4-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "445",
        "kind": "binary",
        "name": "sssd-common",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5692097f24a3b60b6aaf538440af67281eb5997f82bbc487c41026c2f0e9d24e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "475",
        "kind": "binary",
        "name": "prefixdevname",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0d02363e3f6b6e0b69303b4b934fa90f9e9cdbe4036b1ea5f81c20df78d4cf89&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "prefixdevname",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.1.0-8.el9"
        },
        "version": "0.1.0-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "34",
        "kind": "binary",
        "name": "sqlite-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A822659da167b70e82afd5a63f9053a5195e0e23777a143d0fd3852c1f4254f2a",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sqlite",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.34.1-8.el9_6"
        },
        "version": "3.34.1-8.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "422",
        "kind": "binary",
        "name": "perl-IO",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7770e568018ac9d7389628157018827b9f4adafd3988ae02819e3af6b3a13ff5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.43-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "478",
        "kind": "binary",
        "name": "sg3_utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab9b6be4a96fe2f5094bae01a591d20222cd987e7a69e20c30a246b63a60b8bcd&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sg3_utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.47-10.el9"
        },
        "version": "1.47-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "161",
        "kind": "binary",
        "name": "oniguruma",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2ac2cc74f0d5ce457a2531afb412a47db5bc7e3bab4cdc074777caeddd571a5f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "oniguruma",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.9.6-1.el9.6"
        },
        "version": "6.9.6-1.el9.6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "178",
        "kind": "binary",
        "name": "kbd-misc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae2439d682828118a3b5a0c5ac4a2e37274952335e8e74c1f18fe286696f3ce2b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kbd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.4.0-11.el9"
        },
        "version": "2.4.0-11.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "220",
        "kind": "binary",
        "name": "grub2-common",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A203645b66f0c3713afe5ce613699158ae36d712252d2d5d0914509898c1004a3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grub2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.06-104.el9_6"
        },
        "version": "1:2.06-104.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "262",
        "kind": "binary",
        "name": "dbus-tools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab78ae345560559881a3b41fe866e720c28eeb19f87f76b49c73ae9ec5ac42ecb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dbus",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.12.20-8.el9"
        },
        "version": "1:1.12.20-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "274",
        "kind": "binary",
        "name": "sssd-nfs-idmap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab7d682f58fc836c091a6305174eb0a0462346f1cdfeeed8b96ca29007b4dcc19&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "382",
        "kind": "binary",
        "name": "python3-pyserial",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A74cc9c5cf295ec6bf9323709a66a5ecfd99d6bfde71fa0dc5428c2224197e103&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pyserial",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.4-12.el9"
        },
        "version": "3.4-12.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "32",
        "kind": "binary",
        "name": "libcap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6d4e608d43b22200999e79cd5937eb7627faa089415de504341b057b288c3336&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libcap",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.48-9.el9_2"
        },
        "version": "2.48-9.el9_2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "121",
        "kind": "binary",
        "name": "libpcap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae3c699cb1d37a3dde495a0010b7047ee72fc972a8b744d5e6895a2ff90a582dc&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libpcap",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.10.0-4.el9"
        },
        "version": "14:1.10.0-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "167",
        "kind": "binary",
        "name": "libfastjson",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A81a296ac76281feaf611fb6e1ae6ea66b10dfc0dcfd869709b898f806f2fe8f2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libfastjson",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.99.9-5.el9"
        },
        "version": "0.99.9-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "242",
        "kind": "binary",
        "name": "libbabeltrace",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9cae8856532059d637c3658fc0adc5cd70fb67caddf99ef86a52ab130e754b7e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "babeltrace",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.8-10.el9"
        },
        "version": "1.5.8-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "294",
        "kind": "binary",
        "name": "grub2-tools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2c2a9f29365f2c32690a40d7f585c3c27af5f4e86a4706f5c40957316df80aaa&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grub2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.06-104.el9_6"
        },
        "version": "1:2.06-104.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "345",
        "kind": "binary",
        "name": "python3-setools",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A961444f42148f5b5489146bb278bb1b7d13538ec14dd5930f61c3128e7fe6a45&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "setools",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.4.4-1.el9"
        },
        "version": "4.4.4-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "451",
        "kind": "binary",
        "name": "cloud-init",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8547128dc1acef0421aeda11458ab80aab67b5c7a52638adc4ba605650032b10&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cloud-init",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "24.4-4.el9_6.3"
        },
        "version": "24.4-4.el9_6.3"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "333",
        "kind": "binary",
        "name": "python3-libdnf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae2c9e19dd5a18c1657dfad74b5eec6e7c1215c4046bc66856f76180172feb755&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libdnf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.69.0-13.el9"
        },
        "version": "0.69.0-13.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "404",
        "kind": "binary",
        "name": "perl-Class-Struct",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A58c30c37eb85fc976eda1c1c8d1d7367db3987c1c189e729df77e06263aa149e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "0.66-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "465",
        "kind": "binary",
        "name": "NetworkManager-tui",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa61fa6b061f9928271bb99e18990e7804b5e4d1c8bdefa6938f6cab1da2d6cce",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "NetworkManager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.52.0-7.el9_6"
        },
        "version": "1:1.52.0-7.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "102",
        "kind": "binary",
        "name": "libsss_idmap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aed5d6d42a149e1dc3388fec791bc16456fc4d1859a892cdc3e495c54828f874d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "182",
        "kind": "binary",
        "name": "openssl-fips-provider-so",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aeeafdd6863ba7f0fd7e8cfaaed64fbd931edd1e385f6459938e48dacfb1fe6bc&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openssl-fips-provider",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.0.7-6.el9_5"
        },
        "version": "3.0.7-6.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "289",
        "kind": "binary",
        "name": "kernel-modules",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1957f333d072d8c2b5105b2f347b1b532eb5d1d1339d9f1ad6489f0af4298a46",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.14.0-570.41.1.el9_6"
        },
        "version": "5.14.0-570.41.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "337",
        "kind": "binary",
        "name": "python3-systemd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7ceed3e90955bfebdeffdbbdc8d45441a8537d8f7796eecd6562cf2aec208465&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-systemd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "234-19.el9"
        },
        "version": "234-19.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "6",
        "kind": "binary",
        "name": "geolite2-country",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3ca124bcf41a1b098ab969e2a357c1382418a6b211c6e372fc87b53e82bf7ee6&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "geolite2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20191217-6.el9"
        },
        "version": "20191217-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "185",
        "kind": "binary",
        "name": "coreutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0a5b51274b85083cdd7b8e288c021dec0551785852a8a9664c59ca10ebca957d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "coreutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.32-39.el9"
        },
        "version": "8.32-39.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "232",
        "kind": "binary",
        "name": "grub2-pc-modules",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A117c0d11f2abe589b297bcf0f7b5d2a54270e8c04178bc9b799d98e4ea5d6858&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "grub2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.06-104.el9_6"
        },
        "version": "1:2.06-104.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "13",
        "kind": "binary",
        "name": "filesystem",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A92b4b4b744937621e19e953d5e9cb3ff42134fe35ae30b6213334aa8336a4d6d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "filesystem",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.16-5.el9"
        },
        "version": "3.16-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "280",
        "kind": "binary",
        "name": "tpm2-tss",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9766dd6453b237867502d2e6216438e5fb03e778d58dc154eb1c3995a66e8dd2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "tpm2-tss",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.2.3-1.el9"
        },
        "version": "3.2.3-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "296",
        "kind": "binary",
        "name": "fwupd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4215a40b17d6bd89619b33c6b11c9b5a8b3bcbe1baf6bc2e41fd1a3996efd9ae&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "fwupd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.9.26-1.el9"
        },
        "version": "1.9.26-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "403",
        "kind": "binary",
        "name": "perl-Mozilla-CA",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1eb81567e0ccdbdd5f72abd99bb5e03e3d71f4bcc77bea5f0be1af2d88a25336&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Mozilla-CA",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20200520-6.el9"
        },
        "version": "20200520-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "419",
        "kind": "binary",
        "name": "perl-Fcntl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A58a7ec8fe805980046b9f10e17afd2204c2a2c2f2285218ed97a3d1d17ebef24&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.13-481.1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "432",
        "kind": "binary",
        "name": "perl-overload",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abce27c3207f5b38f745548525052591b71f46a3cc6a164d547df031ab4709c41&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.31-481.1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "18",
        "kind": "binary",
        "name": "publicsuffix-list-dafsa",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae90fb7e3f828ea1a88966cb03489850d82a7276cc3b84303a05bda18f1e2b4b2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "publicsuffix-list",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20210518-3.el9"
        },
        "version": "20210518-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "52",
        "kind": "binary",
        "name": "libffi",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A08f483644fb2aa23bd45496e97669262ed1c0b52a22c5e66f450c7357c8b6bec&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libffi",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.4.2-8.el9"
        },
        "version": "3.4.2-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "127",
        "kind": "binary",
        "name": "gdisk",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A994e06c8bcbfeb77cc47582793c4d06ef72a87fa74723f0f7847311f8a0e726e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gdisk",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.7-5.el9"
        },
        "version": "1.0.7-5.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "394",
        "kind": "binary",
        "name": "perl-URI",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af0a82a14c235ee6a6d0cf6316d559e264daed750b87c5c6f5dd8f0f5b0b42bb0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-URI",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.09-3.el9"
        },
        "version": "5.09-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "459",
        "kind": "binary",
        "name": "rpm-plugin-selinux",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0f37ce5b5c1f91b09259bf6fc04ff027337869c81afeb38c994e9dd05c5b97cf&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "39",
        "kind": "binary",
        "name": "libcap-ng",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A231c6b9a173af3d9e7a96e864a51adf162d2c07165c037f079c2dbf05a1ec9cb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libcap-ng",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.8.2-7.el9"
        },
        "version": "0.8.2-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "116",
        "kind": "binary",
        "name": "acl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A75611920b48d4fe48aa5e28387d2dff5c703bbd648854cd7cae0f3bf35065e91&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "acl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.3.1-4.el9"
        },
        "version": "2.3.1-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "365",
        "kind": "binary",
        "name": "python3-cloud-what",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A17e886b40efa459cc7b156537c1ebf31d98ac440b77cf130fb856f014bfb58a2&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "subscription-manager",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.29.45.1-1.el9_6"
        },
        "version": "1.29.45.1-1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "383",
        "kind": "binary",
        "name": "python3-pytz",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af082ffd58b81989b1cbb9519de758564df1b4258ad317fb1b642524f3f2bbb82&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pytz",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2021.1-5.el9"
        },
        "version": "2021.1-5.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "505",
        "kind": "binary",
        "name": "libatomic",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A19a9cc9144bd0f98bea5afa6db8ff9b725b90ae24466a7c70120cc6703a8a507&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gcc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "11.5.0-5.el9_5"
        },
        "version": "11.5.0-5.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "94",
        "kind": "binary",
        "name": "libbasicobjects",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A30edd858fde7d5317393004d4a14b8be5ef3608a5bc1457876c8d3ff6e327dee&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ding-libs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.1-53.el9"
        },
        "version": "0.1.1-53.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "171",
        "kind": "binary",
        "name": "libstemmer",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5d0624a064e50d52f983f95f32b27d65727429265ec49d300f3a59ca23aa6142&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libstemmer",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0-18.585svn.el9"
        },
        "version": "0-18.585svn.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "300",
        "kind": "binary",
        "name": "rpm",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abbc6957f8bea24c20f7d0dd6f8dd2afe3d18121d208c7af81175809d1802a845&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "335",
        "kind": "binary",
        "name": "cockpit-bridge",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8ea1eb53b7a5efcbfbef8cb660db2e5bda9f3c2cdc9c466ec603ee86ec44895c",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cockpit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "334.2-1.el9_6"
        },
        "version": "334.2-1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "56",
        "kind": "binary",
        "name": "libidn2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A64f1e074eb29288ab530aa12586c40c4cfb6f57335348146103ecc0b1e687291&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libidn2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.3.0-7.el9"
        },
        "version": "2.3.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "72",
        "kind": "binary",
        "name": "file",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab59494066384d6261e058b2c4703e5dac9426e7ff7231280d9a58ddd58d232b9&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "file",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.39-16.el9"
        },
        "version": "5.39-16.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "384",
        "kind": "binary",
        "name": "python3-babel",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0676a329800662a626201b403a6d55d55d7231397fe9d6bf5cd114499af9434d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "babel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.1-2.el9"
        },
        "version": "2.9.1-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "425",
        "kind": "binary",
        "name": "perl-Errno",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6385403921d715994c0ce24244874bf104e7d47248bd18c5dcad8101756af14d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.30-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "31",
        "kind": "binary",
        "name": "libxcrypt",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A144a0b4e0c698c12260ef56135a3131f0f4022802a65a2f2b2fea1d451aeac1f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libxcrypt",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.4.18-3.el9"
        },
        "version": "4.4.18-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "49",
        "kind": "binary",
        "name": "dmidecode",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad4ca3702184ef375fdbccfc66374124b33b0112e6d1f17dc2020d3da8bf3f04d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dmidecode",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-1.el9"
        },
        "version": "1:3.6-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "218",
        "kind": "binary",
        "name": "dbus-common",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5c107030c6ee39a810477b48669ed1aa63fd8fe38988d2825bde108d310e1d38&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dbus",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.12.20-8.el9"
        },
        "version": "1:1.12.20-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "251",
        "kind": "binary",
        "name": "quota",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9b9535133b4386f189a487f73f8c52753c1fbba94f4effff58a7229079aa5438&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "quota",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.09-4.el9"
        },
        "version": "1:4.09-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "449",
        "kind": "binary",
        "name": "sssd-kcm",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A95011d8ddd2c99cb463e1ca845c71b7c5973c1902d62ee1fab42fdf5a9d6f182&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "222",
        "kind": "binary",
        "name": "polkit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A52af425fbe71fced1182081105afe86604134e0946043fa97c920093448b2300&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "polkit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.117-13.el9"
        },
        "version": "0.117-13.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "315",
        "kind": "binary",
        "name": "rsyslog-logrotate",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A78b114fc1842657893e260ffddbdd29cb8d812fc72104ea42f1530ce64c34a35&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rsyslog",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.2412.0-1.el9"
        },
        "version": "8.2412.0-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "361",
        "kind": "binary",
        "name": "python3-pexpect",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae6282f3dcd987cbe77210fc93291e8dd1d474af8224b431d62790006f51c1df8&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-pexpect",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.8.0-7.el9"
        },
        "version": "4.8.0-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "28",
        "kind": "binary",
        "name": "xz-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A61d3f2a00f21ccf3968deb03f99cf509a2243388a0e93d97cae9fb0674ffc980&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "xz",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.2.5-8.el9_0"
        },
        "version": "5.2.5-8.el9_0"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "207",
        "kind": "binary",
        "name": "libusbx",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad208e104dce3f28ab67e57653036831d87d89b42a750d23bb98ec897a2caedb7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libusbx",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.0.26-1.el9"
        },
        "version": "1.0.26-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "233",
        "kind": "binary",
        "name": "authselect-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A511d2f13adbf7d3cb1cafdcc53b0834a60b77b6653f2f7ee51f7a59cabe8929a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "authselect",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.2.6-3.el9"
        },
        "version": "1.2.6-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "352",
        "kind": "binary",
        "name": "python3-gpg",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Abbda61538761c292d86aca39d394643584b819413c34e635c32b204eb7d17f67&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gpgme",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.15.1-6.el9"
        },
        "version": "1.15.1-6.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "356",
        "kind": "binary",
        "name": "dnf-plugins-core",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9d55d80d3f5cf2973ddfb837de4378b6c4eb2a5b756c6b9e92eba6d23fdd3f1e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dnf-plugins-core",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.3.0-20.el9"
        },
        "version": "4.3.0-20.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "438",
        "kind": "binary",
        "name": "perl-NDBM_File",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A87a71705dfb33945d633d7a54c8e3aeae03a97c2bb409197005c40f84da15522&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.15-481.1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "7",
        "kind": "binary",
        "name": "geolite2-city",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8d242228d14f7a4bc95c0d25df8b5b81a875064a5fba04cd9d802f7d26a49ad7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "geolite2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20191217-6.el9"
        },
        "version": "20191217-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "229",
        "kind": "binary",
        "name": "openssh",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A496731b3b69460fe55c97a5b87621844cb2f6e51126ca235941d6859d94fdaf3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "openssh",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.7p1-45.el9"
        },
        "version": "8.7p1-45.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "474",
        "kind": "binary",
        "name": "initscripts-rename-device",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5c78e94ded9d958a2d8bc6ac1ea99dd1aa57803f190141c85bd8cae76dff8a69&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "initscripts",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "10.11.8-4.el9"
        },
        "version": "10.11.8-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "477",
        "kind": "binary",
        "name": "tcpdump",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7fb53f145cf8de4997a70fb352f124b142d6adecb0ebcc3cd5f8f56161c694e0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "tcpdump",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.99.0-9.el9"
        },
        "version": "14:4.99.0-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "21",
        "kind": "binary",
        "name": "glibc-gconv-extra",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab79c25eda1e0ddbbbfd4a6cc39ec55140029f55f12dbea2cacc1f17ebd7008ab",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "glibc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.34-168.el9_6.23"
        },
        "version": "2.34-168.el9_6.23"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "84",
        "kind": "binary",
        "name": "ethtool",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac96fb9ef195bef7cc70b5faee01faeefebcdc00740aae28e24b051c101e248ec&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ethtool",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.11-1.el9"
        },
        "version": "2:6.11-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "93",
        "kind": "binary",
        "name": "jansson",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac87a6bf72925c88feebf4e999a88d5e7f7afc92bd0cc9682255e3e9210710459&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "jansson",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.14-1.el9"
        },
        "version": "2.14-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "243",
        "kind": "binary",
        "name": "gssproxy",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aec6e209f8b9e62e94f6e3a4af74f9d1bf8f5cba7e986b081757216d0748c41ed&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gssproxy",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.8.4-7.el9"
        },
        "version": "0.8.4-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "249",
        "kind": "binary",
        "name": "kbd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1d920111139fb61e0862eaa2b2c1f107c1d1a298289b526742bcf8b367b152e3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kbd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.4.0-11.el9"
        },
        "version": "2.4.0-11.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "372",
        "kind": "binary",
        "name": "cockpit-system",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A9df670202e4e868f965b16efa8b973762f736b1e4e9ae9292eac210824d90f9f",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cockpit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "334.2-1.el9_6"
        },
        "version": "334.2-1.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "437",
        "kind": "binary",
        "name": "perl-Exporter",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8467117d279132e28b048c42e13040f76ccb7afb76ea3160ce8d1e0979e07dd3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Exporter",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.74-461.el9"
        },
        "version": "5.74-461.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "53",
        "kind": "binary",
        "name": "libnl3",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0a5f18bdc3e5b8ff349e4325c1c3f280b4559eb2a679922b9f3c6e009dcfed74&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libnl3",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.11.0-1.el9"
        },
        "version": "3.11.0-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "122",
        "kind": "binary",
        "name": "libnl3-cli",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Acc73c45e859787bac2484b188f80b8b6fa912caa54cdf2a61dd0a0d1c888080c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libnl3",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.11.0-1.el9"
        },
        "version": "3.11.0-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "164",
        "kind": "binary",
        "name": "userspace-rcu",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae23a230d62007b9e3dd2e3986631919c357b731c2229a9bc6a6965cfc76aab81&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "userspace-rcu",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.12.1-6.el9"
        },
        "version": "0.12.1-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "272",
        "kind": "binary",
        "name": "passwd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad8468df53b9457b96ce05e9c581ac0ea04944bc9eae308a256b79a8c9841ce98&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "passwd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.80-12.el9"
        },
        "version": "0.80-12.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "351",
        "kind": "binary",
        "name": "python3-decorator",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A371eaed7d532e8d0b970088eb336777175b2e9db048d4ea5cfffa129780a7a74&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-decorator",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.4.2-6.el9"
        },
        "version": "4.4.2-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "473",
        "kind": "binary",
        "name": "man-db",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5714f869d6bdc8cbe4be46b564f580d00c6a6ab68c055708cee44b9d538a995f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "man-db",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.3-7.el9"
        },
        "version": "2.9.3-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "498",
        "kind": "binary",
        "name": "gnutls",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ada23878573f17a459aa0052cd3a6885b6f51e209c09547913ea127933c382c18&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gnutls",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "3.8.3-6.el9_6.2"
        },
        "version": "3.8.3-6.el9_6.2"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "358",
        "kind": "binary",
        "name": "setroubleshoot-plugins",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adf0ad94cfed2420ff5ab48207438d9ac43256e7eef240fe93d1b89c728b0dcfb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "setroubleshoot-plugins",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.3.14-4.el9"
        },
        "version": "3.3.14-4.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "452",
        "kind": "binary",
        "name": "subscription-manager-cockpit",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A42c89f87efdedac9895efdde73b6da537c914453d0b6603306dec4d396f61ac6&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "subscription-manager-cockpit",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6-2.el9"
        },
        "version": "6-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "481",
        "kind": "binary",
        "name": "vim-minimal",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad85d9c2826214b4064407cbf9087213ad711855ce6a7b97877766eeaa5850a1d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "vim",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.2.2637-22.el9_6"
        },
        "version": "2:8.2.2637-22.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "270",
        "kind": "binary",
        "name": "libnfsidmap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A97a3a00216dda4852eb0fc2da00d14e9f27e8b5c2dfeb113c464349d5c8538e7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "nfs-utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5.4-34.el9"
        },
        "version": "1:2.5.4-34.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "319",
        "kind": "binary",
        "name": "python3",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5d24c336c395c46f279134f6bc7a798a3c6a18a575f8696ef2a0760d89353b47",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python3.9",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.9.21-2.el9_6.2"
        },
        "version": "3.9.21-2.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "503",
        "kind": "binary",
        "name": "libldb",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2d520f7abd188b091bda97bb9c658475797301164d8d72249dea1477d681162a&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "samba",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "4.21.3-14.el9_6"
        },
        "version": "4.21.3-14.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "89",
        "kind": "binary",
        "name": "groff-base",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adb1687686e8a5a26586f20668194dec70473bab6b70b50d1a8f95a6e9e17a5f0&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "groff",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.22.4-10.el9"
        },
        "version": "1.22.4-10.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "210",
        "kind": "binary",
        "name": "libgusb",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4bcb98a441f0163f7c59cf4633acb7041ceae75b31489e5094bf8b4c47157dc4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libgusb",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.3.8-2.el9"
        },
        "version": "0.3.8-2.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "213",
        "kind": "binary",
        "name": "pam",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad6585dbd69607b1b2cd1127e3523254dbc789b9a868a18af432fd9b36d881596",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pam",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.1-26.el9_6"
        },
        "version": "1.5.1-26.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "288",
        "kind": "binary",
        "name": "binutils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab8c4f16cc308c68e07cd3e1fc98817bc544e2a7ae8457831fdf27ec267237eee&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "binutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.35.2-63.el9"
        },
        "version": "2.35.2-63.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "354",
        "kind": "binary",
        "name": "dnf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A405e09174999f62ba7d849cddb9bcb05b5db1389be94da8a38ec4861bf7dc16a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dnf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.14.0-25.el9"
        },
        "version": "4.14.0-25.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "427",
        "kind": "binary",
        "name": "perl-Getopt-Std",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A54d0ce3619e6fd2f19aca8b124236704ce55cec2334b333e02fce01279e2aa39&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.32.1-481.1.el9_6"
        },
        "version": "1.12-481.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "480",
        "kind": "binary",
        "name": "efibootmgr",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A90398c529af0a165bc32234872b8df2e55c7b0b97fe9bd2a97d1233a2f2f2c38&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "efibootmgr",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "16-12.el9"
        },
        "version": "16-12.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "177",
        "kind": "binary",
        "name": "dnf-data",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Afbbb48b54c300cb901f2114611249af5d479582c012d4906b18a7726c12e401c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dnf",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.14.0-25.el9"
        },
        "version": "4.14.0-25.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "350",
        "kind": "binary",
        "name": "python3-chardet",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A453a41f3c873e1b2e8f62cd5baa8467b00d7012646ee993643d8a2734ade4a6c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-chardet",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.0.0-5.el9"
        },
        "version": "4.0.0-5.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "355",
        "kind": "binary",
        "name": "python3-dnf-plugins-core",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A193494e5f5858513eb572a7a3c4dbdac1a1ee1b9905e624965ff11350dd9d7fb&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "dnf-plugins-core",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.3.0-20.el9"
        },
        "version": "4.3.0-20.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "188",
        "kind": "binary",
        "name": "krb5-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad3bb2cf83380a9d2f771085f9d017883aed7b54f5cc839cf4fd8782516fe826b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "krb5",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.21.1-8.el9_6"
        },
        "version": "1.21.1-8.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "201",
        "kind": "binary",
        "name": "gzip",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Af13703bcb91ec459b6a0a9227c1456f147b3abf0e9bae8501c7014f9cc4ec71b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gzip",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.12-1.el9"
        },
        "version": "1.12-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "211",
        "kind": "binary",
        "name": "cracklib-dicts",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad521a5a990f2c461389ade5690fbcda86a37be83d8645bd4e10cf253be68be32&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cracklib",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-27.el9"
        },
        "version": "2.9.6-27.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "470",
        "kind": "binary",
        "name": "microcode_ctl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A87e81abed227a0b4957307888f668fccaeebce9899c32fc9739c6963cbe39d7d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "microcode_ctl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20250211-1.20250512.1.el9_6"
        },
        "version": "4:20250211-1.20250512.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "472",
        "kind": "binary",
        "name": "cloud-utils-growpart",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2f84a7157b67f952bdae02724d0905ef938a0f9066c5ba0e88110e3fbec6a108&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cloud-utils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.33-1.el9"
        },
        "version": "0.33-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "303",
        "kind": "binary",
        "name": "selinux-policy",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A623486d1da1a82743c5e650181a7fb93227664582a1321a77122ec74f8b045c4&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "selinux-policy",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "38.1.53-5.el9_6"
        },
        "version": "38.1.53-5.el9_6"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "410",
        "kind": "binary",
        "name": "perl-Term-Cap",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab10e2473a3dc6971cd1caf9c9ee55c6ae74091ab99b290608a019dc760907156&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Term-Cap",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.17-460.el9"
        },
        "version": "1.17-460.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "10",
        "kind": "binary",
        "name": "redhat-release-eula",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6ef0161858cdf00f43bc921169e62468495b57a204967df7b85b83e7892e1184&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "redhat-release",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "9.6-0.1.el9"
        },
        "version": "9.6-0.1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "313",
        "kind": "binary",
        "name": "rpm-sign-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A10ec4e11b713f4b61ab865203c4fdb0af94db5d49a05e9a4a6d366652e8be984&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rpm",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.1.3-37.el9"
        },
        "version": "4.16.1.3-37.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "424",
        "kind": "binary",
        "name": "perl-Pod-Usage",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A530e0e34caac8b2bda7b1bc66f1baec415ccc5fb54a46262e56efe3ceef581af&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Pod-Usage",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.01-4.el9"
        },
        "version": "4:2.01-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "476",
        "kind": "binary",
        "name": "rsync",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab8a2c74b07251f3f2fa3a66fac01be74c70ec045a6e28ea54f4bb37236b8c392&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "rsync",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.2.5-3.el9"
        },
        "version": "3.2.5-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "44",
        "kind": "binary",
        "name": "libgpg-error",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A59ccce61b2b63c7d44d9f4d7913b2f348d8d205c9fd4854c53c0fe058c7a2b5b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libgpg-error",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.42-5.el9"
        },
        "version": "1.42-5.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "135",
        "kind": "binary",
        "name": "systemd-rpm-macros",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5a7838f06c28e677a93b11570c0069beae19b1d545bc6604dc0aa7f9d7d9b66b",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "systemd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "252-51.el9_6.2"
        },
        "version": "252-51.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "492",
        "kind": "binary",
        "name": "libstdc++",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A409a591823148a15e0be6a635f27df4bed65e181ae381eeb7563ac8b14172307&key=199e2f91fd431d51&repoid=rhel-9-for-x86_64-baseos-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gcc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "11.5.0-5.el9_5"
        },
        "version": "11.5.0-5.el9_5"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "128",
        "kind": "binary",
        "name": "libxcrypt-compat",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A785d86d845013993f5306beddf187048b8d7aa7e0559a3bbc22af3960a1a6bd5&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libxcrypt",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.4.18-3.el9"
        },
        "version": "4.4.18-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "131",
        "kind": "binary",
        "name": "libpng",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac8918ffa2cb7a21bb47abe952705746b63132f5a6a532059ce93ca8a504c6df7&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libpng",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.6.37-12.el9"
        },
        "version": "2:1.6.37-12.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "137",
        "kind": "binary",
        "name": "c-ares",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A79e206da92a0a69fbf386c149726881fd2ba9540486491f139a97cff5ad7dcfc&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "c-ares",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.19.1-2.el9_4"
        },
        "version": "1.19.1-2.el9_4"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "174",
        "kind": "binary",
        "name": "linux-firmware",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A8ce476a6b0ec0c18b9ade009f8abcb277321b97f898c770390f333af57aa5a25",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "linux-firmware",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "20250812-151.4.el9_6"
        },
        "version": "20250812-151.4.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "214",
        "kind": "binary",
        "name": "util-linux",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5c5eb489cc56d6ebadbc84c6aa0e419f6da8f544c4c8dbcc9710991425853b62&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "util-linux",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.37.4-21.el9"
        },
        "version": "2.37.4-21.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "244",
        "kind": "binary",
        "name": "libkcapi",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa0a77df9c5e95e1d7e5b7e2ec903441d753ab5ccf0b41b8cf4c6cbd273f85c1a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libkcapi",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.4.0-2.el9"
        },
        "version": "1.4.0-2.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "360",
        "kind": "binary",
        "name": "python3-ptyprocess",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0a113f08cf9774e5b20312a9efa0ceb76387f21e9dfe83a7031e32e97e3b5d0f&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "python-ptyprocess",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.6.0-12.el9"
        },
        "version": "0.6.0-12.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "400",
        "kind": "binary",
        "name": "perl-Net-SSLeay",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A0bb2df441047fe0466861b2533b19c3bd72b8110ad9e15062416fff4d41eb120&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "perl-Net-SSLeay",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.94-1.el9"
        },
        "version": "1.94-1.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "0",
        "kind": "binary",
        "name": "tzdata",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A500c2c58a97662923434418e5208e0677e48474cfc45031f1857acd24e6a4a26&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "tzdata",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2025b-1.el9"
        },
        "version": "2025b-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "58",
        "kind": "binary",
        "name": "libattr",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab22bffd5f8bad528f6293f8975160a15187ddd56626fe85f689bdae2c4544e5c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "attr",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5.1-3.el9"
        },
        "version": "2.5.1-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "62",
        "kind": "binary",
        "name": "pcre2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A6d80172d46ad12037cd7210009286e9e4077623762dffcc8ea24e99dcfca530a&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "pcre2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "10.40-6.el9"
        },
        "version": "10.40-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "136",
        "kind": "binary",
        "name": "ncurses",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A907eaf14c7b80fdbe7755012c9886f3fa26c97c0ef79c76c4d4f26bdf16ed17b&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ncurses",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.2-10.20210508.el9_6.2"
        },
        "version": "6.2-10.20210508.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "68",
        "kind": "binary",
        "name": "libtevent",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A46f02d78d5365ebdd87a1d670d263ba4910694f6dbf43230883eaca48905c395&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libtevent",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.16.1-1.el9"
        },
        "version": "0.16.1-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "74",
        "kind": "binary",
        "name": "psmisc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae03fa1e408247ff2de85c63398b32025d6c7d2252b5708a0e1a374439d7b0562&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "psmisc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "23.4-3.el9"
        },
        "version": "23.4-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "92",
        "kind": "binary",
        "name": "fuse-libs",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A3b800511ee3c23cd49e5a406c28da600a0940636258a2e08dd2a80afe1dd6914&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "fuse",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.9-17.el9"
        },
        "version": "2.9.9-17.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "172",
        "kind": "binary",
        "name": "liburing",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad24f1383eb63751d4233929e010691211834858ebdaedbdc7cd49840721c20fc&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "liburing",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.5-1.el9"
        },
        "version": "2.5-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "109",
        "kind": "binary",
        "name": "xz",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A1190a84cf1adc58b3fd05c51be44002317106465edfe6dd82d493ee0358e80f3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "xz",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "5.2.5-8.el9_0"
        },
        "version": "5.2.5-8.el9_0"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "186",
        "kind": "binary",
        "name": "ca-certificates",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ab2d4c40e1e6689d833e8fa41289b0f06ea6ac64384abca36924f9567dbf3cc71&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "ca-certificates",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2024.2.69_v8.0.303-91.4.el9_4"
        },
        "version": "2024.2.69_v8.0.303-91.4.el9_4"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "250",
        "kind": "binary",
        "name": "sssd-client",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A388a8bb4faae7adc3399928790a481e8a3d99eca028e1f80e3d74e0e003cb012&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "sssd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.9.6-4.el9_6.2"
        },
        "version": "2.9.6-4.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "260",
        "kind": "binary",
        "name": "glib-networking",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A4a939056b72eb86add24482accf98c11071a8a11602c9cf9a0b17ca95be0f9b3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "glib-networking",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.68.3-3.el9"
        },
        "version": "2.68.3-3.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "269",
        "kind": "binary",
        "name": "gpgme",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A76257ff03c0818895a63df0acdf622382cf83b3a6b13ccfe7efcfff2aab8f6b3&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gpgme",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.15.1-6.el9"
        },
        "version": "1.15.1-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "499",
        "kind": "binary",
        "name": "python3-perf",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A68d99d5231e3f172138b67e49e8dc0bc4c740a35cc492eec5b5afa0ea7fe9833&repoid=rhel-9-for-x86_64-appstream-rpms",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "kernel",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "repositoryHint": "repoid=rhel-9-for-x86_64-appstream-rpms&repoid=rhel-9-for-x86_64-baseos-rpms",
          "version": "5.14.0-570.49.1.el9_6"
        },
        "version": "5.14.0-570.49.1.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "78",
        "kind": "binary",
        "name": "libtasn1",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A5f26f07fd500969d1657a07bccbde430ad59d1be72ee37bfd0ae7a23f7add206&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libtasn1",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "4.16.0-9.el9"
        },
        "version": "4.16.0-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "141",
        "kind": "binary",
        "name": "inih",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ae2d56e1fc412201280348ab8f2ac31782444cda5ba497695a09ca801104a5459&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "inih",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "49-6.el9"
        },
        "version": "49-6.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "217",
        "kind": "binary",
        "name": "systemd",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A162b568bc4361aee87e3fc538a18bac49f38528d25fd9c9e62a7f935946e446a",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "systemd",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "252-51.el9_6.2"
        },
        "version": "252-51.el9_6.2"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "225",
        "kind": "binary",
        "name": "cronie-anacron",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ad72c1e2f3bafea6f35ccb17f3f52bf4958edad5c18e4443c46864d08e40d3e8c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "cronie",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.5.7-14.el9_6"
        },
        "version": "1.5.7-14.el9_6"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "268",
        "kind": "binary",
        "name": "gnupg2",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A33c4b71816b08138fb21302fff14e7e981c98294817c9aae66fe4f1afdcb0979&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gnupg2",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "2.3.3-4.el9"
        },
        "version": "2.3.3-4.el9"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "14",
        "kind": "binary",
        "name": "efi-filesystem",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7d3214f9908354a7b8f97bb3d2b3b951c775c36ace0f45306b5c7a60c462de0c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "efi-rpm-macros",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6-2.el9_0"
        },
        "version": "6-2.el9_0"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "118",
        "kind": "binary",
        "name": "gettext",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A2cc426af0adb9eaa2d5285b5d2eaff049cf903ca85242b7739e87e84397bd1da&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "gettext",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.21-8.el9"
        },
        "version": "0.21-8.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "479",
        "kind": "binary",
        "name": "iproute-tc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A7f11b0b3001da6e6bb561b2bda4e9b9ef535bf4b7ecb21ce8a9cfb91e191641d&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "iproute",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "6.11.0-1.el9"
        },
        "version": "6.11.0-1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "37",
        "kind": "binary",
        "name": "libcom_err",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A218da3ff688170446710149b0595b48634e09d1cf1de7cbb9c0550a195419532&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "e2fsprogs",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.46.5-7.el9"
        },
        "version": "1.46.5-7.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "43",
        "kind": "binary",
        "name": "readline",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ac1e5ef808e31e35abca640518c27b3e54f0fdae444923da374b0b97a0f46d62c&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "readline",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "8.1-4.el9"
        },
        "version": "8.1-4.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "123",
        "kind": "binary",
        "name": "libteam",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aa4f1026023e9b354c3d8c06b2d83e12734d71771eaa6b67a0671131b9c8c2199&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libteam",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.31-16.el9_1"
        },
        "version": "1.31-16.el9_1"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "200",
        "kind": "binary",
        "name": "libtirpc",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Ace87144fb6f26977e61c335f8e602541ac76a1ae1e343ee47147c5ff5f89ae26&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libtirpc",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "1.3.3-9.el9"
        },
        "version": "1.3.3-9.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "299",
        "kind": "binary",
        "name": "curl",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Adaa9c115558c638582f721bb556bd7ec422854ae2e5ce6cc833a341433bf9888&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "curl",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "7.76.1-31.el9_6.1"
        },
        "version": "7.76.1-31.el9_6.1"
      },
      {
        "arch": "noarch",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "347",
        "kind": "binary",
        "name": "policycoreutils-python-utils",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3Aeebaccdb1aed2c30a64839bb7bedc1d78ae2de203a4e55d5083121031db8687e&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "policycoreutils",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "3.6-2.1.el9"
        },
        "version": "3.6-2.1.el9"
      },
      {
        "arch": "x86_64",
        "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
        "id": "46",
        "kind": "binary",
        "name": "libunistring",
        "normalizedVersion": {
          "v": [
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
          ]
        },
        "packageDb": "sqlite:var/lib/rpm",
        "repositoryHint": "hash=sha256%3A66dae0965774d784f17607063c77fa2612be733a54fc22b2a1b78f744136af35&key=199e2f91fd431d51",
        "source": {
          "cpe": "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
          "kind": "source",
          "name": "libunistring",
          "normalizedVersion": {
            "v": [
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0,
              0
            ]
          },
          "packageDb": "sqlite:var/lib/rpm",
          "version": "0.9.10-15.el9"
        },
        "version": "0.9.10-15.el9"
      }
    ],
    "repositories": [
      {
        "cpe": "cpe:2.3:a:redhat:enterprise_linux:9:*:appstream:*:*:*:*:*",
        "id": "0",
        "key": "rhel-cpe-repository",
        "name": "rhel-9-for-x86_64-appstream-rpms"
      },
      {
        "cpe": "cpe:2.3:o:redhat:enterprise_linux:9:*:baseos:*:*:*:*:*",
        "id": "1",
        "key": "rhel-cpe-repository",
        "name": "rhel-9-for-x86_64-baseos-rpms"
      }
    ]
  },
  "state": "IndexFinished",
  "success": true
}`
