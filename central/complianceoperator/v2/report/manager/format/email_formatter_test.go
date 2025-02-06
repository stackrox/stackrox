package format

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestComplianceReportingEmailFormatter(t *testing.T) {
	suite.Run(t, new(ComplianceReportingEmailFormatterSuite))
}

type ComplianceReportingEmailFormatterSuite struct {
	suite.Suite
}

func (s *ComplianceReportingEmailFormatterSuite) Test_FormatWithDetails() {
	templateName := "templateName"
	emailFormatter := NewEmailFormatter()
	type templateData struct {
		Text string
	}
	s.Run("expect error if data is missing a field", func() {
		_, err := emailFormatter.FormatWithDetails(templateName, "template {{.Field}}", &templateData{Text: "text"})
		s.Assert().Error(err)
	})
	s.Run("success", func() {
		tmpl, err := emailFormatter.FormatWithDetails(templateName, "template {{.Text}}", &templateData{Text: "text"})
		s.Assert().Nil(err)
		s.Assert().Equal("template text", tmpl)
		s.T().Log(tmpl)
	})
}
