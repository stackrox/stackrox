package printers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testResultObject struct {
	Result testCSVDataObject `json:"result"`
}

type testCSVDataObject struct {
	Schools  []testSchoolObject `json:"schools"`
	Pupils   []int              `json:"pupils"`
	Teachers []int              `json:"teachers"`
}

type testSchoolObject struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

var (
	testColumnHeaders = []string{"SCHOOL", "PUPILS", "TEACHERS"}
	testRowExpression = "{result.schools.#.name,result.pupils,result.teachers}"
	testCSVObject     = &testResultObject{
		Result: testCSVDataObject{
			Schools: []testSchoolObject{
				{
					Name:    "School1",
					Address: "Address1",
				},
				{
					Name:    "School2",
					Address: "Address2",
				},
				{
					Name:    "School3",
					Address: "Address3",
				},
				{
					Name:    "School4",
					Address: "Address4",
				},
			},
			Pupils:   []int{10, 20, 30, 40},
			Teachers: []int{1, 2, 3, 4},
		},
	}
)

func TestCsvPrinter_Print(t *testing.T) {
	cases := map[string]struct {
		expectedOutput   string
		noHeaders        bool
		headerAsComments bool
	}{
		"no settings specified": {
			expectedOutput: `SCHOOL,PUPILS,TEACHERS
School1,10,1
School2,20,2
School3,30,3
School4,40,4
`,
		},
		"print without headers": {
			expectedOutput: `School1,10,1
School2,20,2
School3,30,3
School4,40,4
`,
			noHeaders: true,
		},
		"print with headers as comments": {
			expectedOutput: `; SCHOOL,PUPILS,TEACHERS
School1,10,1
School2,20,2
School3,30,3
School4,40,4
`,
			headerAsComments: true,
		},
		"when specifying print no headers and print headers as comments - no headers should be printed with precedence": {
			expectedOutput: `School1,10,1
School2,20,2
School3,30,3
School4,40,4
`,
			noHeaders:        true,
			headerAsComments: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := strings.Builder{}
			printer := NewCSVPrinter(testRowExpression, WithCSVColumnHeaders(testColumnHeaders),
				WithCSVHeaderOptions(c.noHeaders, c.headerAsComments))
			err := printer.Print(&testCSVObject, &out)
			require.NoError(t, err)
			assert.Equal(t, c.expectedOutput, out.String())
		})
	}
}

func TestCsvPrinter_Print_Failures(t *testing.T) {
	cases := map[string]struct {
		out            *strings.Builder
		headers        []string
		rowExpression  string
		object         interface{}
		expectedOutput string
	}{
		"invalid JSON should cause a failure": {
			out:            &strings.Builder{},
			object:         make(chan int),
			headers:        testColumnHeaders,
			rowExpression:  testRowExpression,
			expectedOutput: "",
		},
		"jagged input": {
			out:           &strings.Builder{},
			headers:       testColumnHeaders,
			rowExpression: testRowExpression,
			object: &testResultObject{Result: testCSVDataObject{
				Schools: []testSchoolObject{
					{
						Name: "School1",
					},
					{
						Name: "School2",
					},
					{
						Name: "School3",
					},
				},
				Pupils:   []int{10, 20},
				Teachers: []int{1, 2, 3},
			}},
			expectedOutput: "",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := NewCSVPrinter(c.rowExpression, WithCSVColumnHeaders(c.headers), WithCSVHeaderOptions(false, false))
			err := p.Print(c.object, c.out)
			require.Error(t, err)
			assert.Equal(t, c.expectedOutput, c.out.String())
		})
	}
}

func TestCsvPrinter_Print_EmptyData(t *testing.T) {
	cases := map[string]struct {
		out             *strings.Builder
		noHeader        bool
		headerAsComment bool
		expectedOutput  string
	}{
		"empty data should only print headers": {
			out:            &strings.Builder{},
			expectedOutput: "SCHOOL,PUPILS,TEACHERS\n",
		},
		"empty data with header as comment should print commented headers": {
			out:             &strings.Builder{},
			expectedOutput:  "; SCHOOL,PUPILS,TEACHERS\n",
			headerAsComment: true,
		},
		"empty data with no header set should print nothing": {
			out:            &strings.Builder{},
			expectedOutput: "",
			noHeader:       true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := NewCSVPrinter(testRowExpression, WithCSVColumnHeaders(testColumnHeaders),
				WithCSVHeaderOptions(c.noHeader, c.headerAsComment))
			err := p.Print(&testResultObject{}, c.out)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedOutput, c.out.String())
		})
	}
}

func TestCsvPrinter_ReadCSVOutputWithCommentedHeaders_Success(t *testing.T) {
	p := NewCSVPrinter(testRowExpression, WithCSVColumnHeaders(testColumnHeaders),
		WithCSVHeaderOptions(false, true))
	out := &bytes.Buffer{}
	require.NoError(t, p.Print(testCSVObject, out))
	r := csv.NewReader(out)
	// Since Comment is per default not set, need to set it explicitly
	r.Comment = ';'
	records, err := r.ReadAll()
	require.NoError(t, err)
	for _, record := range records {
		assert.False(t, strings.HasPrefix(strings.Join(record, ","), ";"))
	}
}

func TestCsvPrinter_ReadCSVOutputWithCommentedHeaders_Failure(t *testing.T) {
	p := NewCSVPrinter(testRowExpression, WithCSVColumnHeaders(testColumnHeaders),
		WithCSVHeaderOptions(false, true))
	out := &bytes.Buffer{}
	require.NoError(t, p.Print(testCSVObject, out))
	r := csv.NewReader(out)
	// Since Comment is per default not set, need to set it explicitly
	r.Comment = '#'
	records, err := r.ReadAll()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(strings.Join(records[0], ","), ";"))
}

func TestCsvPrinter(t *testing.T) {
	type vulnObj struct {
		Name string `json:"vulnName"`
	}

	type compObj struct {
		Name  string    `json:"compName"`
		Vulns []vulnObj `json:"vulns"`
	}

	type imgObj struct {
		Name       string    `json:"imgName"`
		Components []compObj `json:"components"`
	}

	type depObj struct {
		Name   string   `json:"depName"`
		Images []imgObj `json:"images"`
	}

	type testObject struct {
		Deployments []depObj `json:"deployments"`
	}
	type resultObject struct {
		Result testObject `json:"result"`
	}

	obj := &resultObject{
		Result: testObject{
			Deployments: []depObj{
				{
					Name: "dep1",
					Images: []imgObj{
						{
							Name: "image1",
							Components: []compObj{
								{
									Name: "comp11",
									Vulns: []vulnObj{
										{
											Name: "cve1",
										},
									},
								},
								{
									Name:  "comp12",
									Vulns: []vulnObj{},
								},
							},
						},
						{
							Name: "image2",
							Components: []compObj{
								{
									Name: "comp21",
									Vulns: []vulnObj{
										{
											Name: "cve1",
										},
									},
								},
								{
									Name: "comp22",
									Vulns: []vulnObj{
										{
											Name: "cve2",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	columnHeaders := []string{
		"Deployment",
		"Image",
		"Component",
		"Vuln",
	}

	rowExpression := "{result.deployments.#.depName,result.deployments.#.images.#.imgName,result.deployments.#.images.#.components.#.compName,result.deployments.#.images.#.components.#.vulns.#.vulnName}"

	p := newCSVPrinter(columnHeaders, rowExpression, false, false)
	out := &bytes.Buffer{}
	require.NoError(t, p.Print(&obj, out))

	fmt.Println(out)
}
