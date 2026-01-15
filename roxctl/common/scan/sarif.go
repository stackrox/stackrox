package scan

import (
	"github.com/stackrox/rox/pkg/gjson"
	"github.com/stackrox/rox/pkg/printers"
)

var (
	// JSON Path expressions to use for sarif report generation
	SarifJSONPathExpressions = map[string]string{
		printers.SarifRuleJSONPathExpressionKey: gjson.MultiPathExpression(
			`@text:{"printKeys":"false","customSeparator":"_"}`,
			gjson.Expression{
				Expression: "result.vulnerabilities.#.cveId",
			},
			gjson.Expression{
				Expression: "result.vulnerabilities.#.componentName",
			},
			gjson.Expression{
				Expression: "result.vulnerabilities.#.componentVersion",
			},
		),
		printers.SarifHelpJSONPathExpressionKey: gjson.MultiPathExpression(
			"@text",
			gjson.Expression{
				Key:        "Vulnerability",
				Expression: "result.vulnerabilities.#.cveId",
			},
			gjson.Expression{
				Key:        "Link",
				Expression: "result.vulnerabilities.#.cveInfo",
			},
			gjson.Expression{
				Key:        "Severity",
				Expression: "result.vulnerabilities.#.cveSeverity",
			},
			gjson.Expression{
				Key:        "CVSS",
				Expression: "result.vulnerabilities.#.cveCvss",
			},
			gjson.Expression{
				Key:        "Component",
				Expression: "result.vulnerabilities.#.componentName",
			},
			gjson.Expression{
				Key:        "Version",
				Expression: "result.vulnerabilities.#.componentVersion",
			},
			gjson.Expression{
				Key:        "Fixed Version",
				Expression: "result.vulnerabilities.#.componentFixedVersion",
			},
			gjson.Expression{
				Key:        "Advisory",
				Expression: "result.vulnerabilities.#.advisoryId",
			},
			gjson.Expression{
				Key:        "Advisory Link",
				Expression: "result.vulnerabilities.#.advisoryInfo",
			},
		),
		printers.SarifSeverityJSONPathExpressionKey: "result.vulnerabilities.#.cveSeverity",
		printers.SarifHelpLinkJSONPathExpressionKey: "result.vulnerabilities.#.cveInfo",
	}
)
