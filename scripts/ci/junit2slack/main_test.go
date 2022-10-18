package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/GoogleCloudPlatform/testgrid/metadata/junit"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	//go:embed testdata/sample.xml
	sample string
)

func TestConstructSlackMessage(t *testing.T) {
	expected := []byte(`[{"color":"#bb2124","blocks":[{"type":"header","text":{"type":"plain_text","text":"Failed tests"}},{"type":"section","text":{"type":"plain_text","text":"CSVTest: Verify CVE CSV data scoped by entity is correct[2]"}}]},{"color":"#bb2124","blocks":[{"type":"section","text":{"type":"mrkdwn","text":"Junit failure message for *CSVTest: Verify CVE CSV data scoped by entity is correct[2]*"}},{"type":"section","text":{"type":"plain_text","text":"Condition not satisfied:\n\ncsvCVEs.get(i) == graphQLCVEs.get(i)\n|       |   |  |  |           |   |\n|       |   41 |  |           |   41\n|       |      |  |           \u003cCSVTest$CVE@5ff1c5e4 id=CVE-2019-12900 cvss=9.8 deploymentCount=1 imageCount=1 componentCount=4 this$0=CSVTest@5d38a487\u003e\n|       |      |  [CSVTest$CVE@ab0b0a83, CSVTest$CVE@49bfa946, CSVTest$CVE@e2b5b6f6, CSVTest$CVE@34c4eb97, CSVTest$CVE@34e11a99, CSVTest$CVE@c7e9a603, CSVTest$CVE@c958091d, CSVTest$CVE@9447532e, CSVTest$CVE@94556ab0, CSVTest$CVE@958b6fc6, CSVTest$CVE@9ccf8e4a, CSVTest$CVE@c51085a4, CSVTest$CVE@41caa49, CSVTest$CVE@a141b245, CSVTest$CVE@711c6105, CSVTest$CVE@712a7886, CSVTest$CVE@3f4d36fc, CSVTest$CVE@683e6e39, CSVTest$CVE@6b9f0ca1, CSVTest$CVE@6bad2422, CSVTest$CVE@6bbb3ba3, CSVTest$CVE@6bc95324, CSVTest$CVE@6bd76aa5, CSVTest$CVE@6be58226, CSVTest$CVE@6d299ebd, CSVTest$CVE@bea6801, CSVTest$CVE@c069703, CSVTest$CVE@d59277db, CSVTest$CVE@d5a08f5c, CSVTest$CVE@dad9b53a, CSVTest$CVE@20186ee4, CSVTest$CVE@2d493efe, CSVTest$CVE@2d57567f, CSVTest$CVE@1f6c0902, CSVTest$CVE@6215383d, CSVTest$CVE@400508c7, CSVTest$CVE@d710f0fb, CSVTest$CVE@fc1cbb1c, CSVTest$CVE@feb1acf5, CSVTest$CVE@ecbda408, CSVTest$CVE@26a6537d, CSVTest$CVE@5ff1c5e4, CSVTest$CVE@eb8a9e66, CSVTest$CVE@c2bc6fb2, CSVTest$CVE@dde6e578, CSVTest$CVE@7c38dfb5, CSVTest$CVE@f7b814ad, CSVTest$CVE@d0053119, CSVTest$CVE@8574866d, CSVTest$CVE@86f10108, CSVTest$CVE@e80ecfed, CSVTest$CVE@e97d3307, CSVTest$CVE@885c184d, CSVTest$CVE@b2753d1f, CSVTest$CVE@190248e6, CSVTest$CVE@f53fa006, CSVTest$CVE@c5ec5080, CSVTest$CVE@c6087f82, CSVTest$CVE@5056cf5d, CSVTest$CVE@212dc659, CSVTest$CVE@2b283e8e, CSVTest$CVE@25d65b57, CSVTest$CVE@fe8dd239, CSVTest$CVE@70945666, CSVTest$CVE@edad99c6, CSVTest$CVE@94a5bfdc, CSVTest$CVE@851d1321, CSVTest$CVE@2c2568fb, CSVTest$CVE@eca16279, CSVTest$CVE@e5511974, CSVTest$CVE@e55f30f5, CSVTest$CVE@32589016, CSVTest$CVE@29b612f9, CSVTest$CVE@74f75429, CSVTest$CVE@54f6a8bb, CSVTest$CVE@f5e08943, CSVTest$CVE@a58c86e5, CSVTest$CVE@6edadf49, CSVTest$CVE@3e277299, CSVTest$CVE@3c8a1e51, CSVTest$CVE@4c136df1, CSVTest$CVE@49def44, CSVTest$CVE@4ac06c5, CSVTest$CVE@4ba1e46, CSVTest$CVE@34ba2dae, CSVTest$CVE@65fe59ad, CSVTest$CVE@69ae8070, CSVTest$CVE@6ae48586, CSVTest$CVE@6af29d07, CSVTest$CVE@6b00b488, CSVTest$CVE@8de64d7d, CSVTest$CVE@cb1352ff, CSVTest$CVE@cc8fcd9a, CSVTest$CVE@30ed073a, CSVTest$CVE@d17ed81f, CSVTest$CVE@d9c09db5, CSVTest$CVE@d9ceb536, CSVTest$CVE@6aa11951, CSVTest$CVE@68c3e72f, CSVTest$CVE@98615211, CSVTest$CVE@39df66b1, CSVTest$CVE@f2bf931a, CSVTest$CVE@27c1992f, CSVTest$CVE@c5c47329, CSVTest$CVE@acf47c90, CSVTest$CVE@d576e4c5, CSVTest$CVE@9a12044e, CSVTest$CVE@9a4a6252, CSVTest$CVE@9a5879d3, CSVTest$CVE@9a669154, CSVTest$CVE@9a74a8d5, CSVTest$CVE@bf8072f6, CSVTest$CVE@bf8e8a77, CSVTest$CVE@bf9ca1f8, CSVTest$CVE@3fa5bdd8, CSVTest$CVE@71b61e0c, CSVTest$CVE@3fdde12d, CSVTest$CVE@3febf8ae, CSVTest$CVE@1a2a6ff7, CSVTest$CVE@a4a453df, CSVTest$CVE@4bc5f8f8, CSVTest$CVE@5423ed90, CSVTest$CVE@579"}}]}]`)

	var junitFiles []*junit.Suites
	suites, err := junit.Parse([]byte(sample))
	assert.NoError(t, err, "If this fails, it probably indicates a problem with the sample junit report rather than the code")
	assert.NotNil(t, suites, "If this fails, it probably indicates a problem with the sample junit report rather than the code")

	junitFiles = append(junitFiles, suites)
	blocks := convertJunitToSlack(junitFiles)
	b, err := json.Marshal(blocks)
	assert.NoError(t, err)
	fmt.Println(string(b))

	assert.Equal(t, expected, b)
}
