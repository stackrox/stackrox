package sbomer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/quay/claircore/sbom/spdx"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
)

func TestDave(t *testing.T) {
	v4IRB, err := os.ReadFile("/Users/dcaravel/Downloads/sboms/v4-IndexReport.json")
	if err != nil {
		t.Fatal(err)
	}

	// var v4ir *v4.VulnerabilityReport
	v4ir := new(v4.IndexReport)
	err = json.Unmarshal(v4IRB, v4ir)
	if err != nil {
		t.Fatal(err)
	}

	ir, err := mappers.ToClairCoreIndexReport(v4ir.GetContents())
	if err != nil {
		t.Fatal(err)
	}

	// t.Logf("%+v", ir)

	encoder := spdx.Encoder{
		Version: spdx.V2_3,
		Format:  spdx.JSON,
		Creators: []spdx.Creator{
			{Creator: "David Caravello", CreatorType: "Person"},
			{Creator: "Brad Lugo", CreatorType: "Person"},
			{Creator: "David Vail", CreatorType: "Person"},
			{Creator: "Surabhi LNU", CreatorType: "Person"},
		},
		DocumentName:      "hi",
		DocumentNamespace: "DocumentNamespace?",
		DocumentComment:   "Bug Bash Demo",
	}

	ctx := context.Background()
	reader, err := encoder.Encode(ctx, ir)
	if err != nil {
		t.Fatal(err)
	}
	dataB, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, dataB, "", "  ")
	fmt.Printf("%s\n", prettyJSON.String())
}
