package format

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/format/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestComplianceReportingFormatter(t *testing.T) {
	suite.Run(t, new(ComplianceReportingFormatterSuite))
}

type ComplianceReportingFormatterSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	zipWriter *mocks.MockZipWriter
	csvWriter *mocks.MockCSVWriter

	formatter *FormatterImpl
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportNoError() {
	matcher := &valueMatcher{
		data: getFakeReportData(),
	}
	s.zipWriter.EXPECT().Create(gomock.Any()).Times(1).Return(nil, nil)
	s.csvWriter.EXPECT().AddValue(matcher).Times(2).Do(func(_ any) {
		matcher.recordNumber++
	})
	s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(nil)
	s.zipWriter.EXPECT().Close().Times(1).Return(nil)

	buf, err := s.formatter.FormatCSVReport(getFakeReportData(), getFakeEmptyFailedClusters())
	s.Require().NoError(err)
	s.Require().NotNil(buf)
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportWithFailedClusterNoError() {
	matcher := &failedClusterValueMatcher{}
	matcher.data, matcher.failed = getFakeReportDataWithFailedCluster()
	s.zipWriter.EXPECT().Create(gomock.Any()).Times(2).Return(nil, nil)
	s.csvWriter.EXPECT().AddValue(matcher).Times(3)
	s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(2).Return(nil)
	s.zipWriter.EXPECT().Close().Times(1).Return(nil)

	buf, err := s.formatter.FormatCSVReport(getFakeReportDataWithFailedCluster())
	s.Require().NoError(err)
	s.Require().NotNil(buf)
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportCreateError() {
	s.Run("report data", func() {
		s.zipWriter.EXPECT().Create(gomock.Any()).Times(1).Return(nil, errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(getFakeReportData(), getFakeEmptyFailedClusters())
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
	s.Run("failed cluster data", func() {
		s.zipWriter.EXPECT().Create(gomock.Any()).Times(1).Return(nil, errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(getFakeReportDataOnlyFailedCluster())
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportWriteError() {
	s.Run("report date", func() {
		s.zipWriter.EXPECT().Create(gomock.Any()).Times(1).Return(nil, nil)
		s.csvWriter.EXPECT().AddValue(gomock.Any()).Times(2)
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(getFakeReportData(), getFakeEmptyFailedClusters())
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
	s.Run("failed cluster date", func() {
		matcher := &failedClusterValueMatcher{}
		matcher.data, matcher.failed = getFakeReportDataOnlyFailedCluster()
		s.zipWriter.EXPECT().Create(gomock.Any()).Times(1).Return(nil, nil)
		s.csvWriter.EXPECT().AddValue(matcher).Times(1)
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(getFakeReportDataOnlyFailedCluster())
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportCloseError() {
	s.zipWriter.EXPECT().Create(gomock.Any()).Times(1).Return(nil, nil)
	s.csvWriter.EXPECT().AddValue(gomock.Any()).Times(2)
	s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(nil)
	s.zipWriter.EXPECT().Close().Times(1).Return(errors.New("error"))

	buf, err := s.formatter.FormatCSVReport(getFakeReportData(), getFakeEmptyFailedClusters())
	s.Require().Error(err)
	s.Require().Nil(buf)
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportEmptyReportNoError() {
	s.zipWriter.EXPECT().Create(gomock.Any()).Times(1).Return(nil, nil)
	s.csvWriter.EXPECT().AddValue(&emptyValueMatcher{
		value: EmptyValue,
		data:  getFakeEmptyReportData(),
	}).Times(1)
	s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(nil)
	s.zipWriter.EXPECT().Close().Times(1).Return(nil)

	buf, err := s.formatter.FormatCSVReport(getFakeEmptyReportData(), getFakeEmptyFailedClusters())
	s.Require().NoError(err)
	s.Require().NotNil(buf)
}

func (s *ComplianceReportingFormatterSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.zipWriter = mocks.NewMockZipWriter(s.ctrl)
	s.csvWriter = mocks.NewMockCSVWriter(s.ctrl)

	s.formatter = &FormatterImpl{
		newZipWriter: s.getZipWriter(),
		newCSVWriter: s.getCSVWriter(),
	}
}

func (s *ComplianceReportingFormatterSuite) getZipWriter() func(*bytes.Buffer) ZipWriter {
	return func(_ *bytes.Buffer) ZipWriter {
		return s.zipWriter
	}
}

func (s *ComplianceReportingFormatterSuite) getCSVWriter() func(csv.Header, bool) CSVWriter {
	return func(_ csv.Header, _ bool) CSVWriter {
		return s.csvWriter
	}
}

func getFakeEmptyReportData() map[string][]*report.ResultRow {
	results := make(map[string][]*report.ResultRow)
	results["cluster-1"] = []*report.ResultRow{}
	return results
}

func getFakeReportData() map[string][]*report.ResultRow {
	results := make(map[string][]*report.ResultRow)
	results["cluster-1"] = []*report.ResultRow{
		{
			ClusterName: "test_cluster-1",
			CheckName:   "test_check-1",
			Profile:     "test_profile-1",
			ControlRef:  "test_control_ref-1",
			Description: "description-1",
			Status:      "Pass",
			Remediation: "remediation-1",
		},
		{
			ClusterName: "test_cluster-1",
			CheckName:   "test_check-2",
			Profile:     "test_profile-2",
			ControlRef:  "test_control_ref-2",
			Description: "description-2",
			Status:      "Fail",
			Remediation: "remediation-2",
		},
	}
	return results
}

func getFakeReportDataWithFailedCluster() (map[string][]*report.ResultRow, map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster) {
	failedClusters := make(map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster)
	failedClusters["cluster-2"] = &storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
		ClusterName:     "test_cluster-2",
		ClusterId:       "test_cluster-2-id",
		Reason:          "timeout",
		OperatorVersion: "v1.6.0",
	}
	results := getFakeReportData()
	results["cluster-2"] = []*report.ResultRow{}
	return results, failedClusters
}

func getFakeReportDataOnlyFailedCluster() (map[string][]*report.ResultRow, map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster) {
	failedClusters := make(map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster)
	failedClusters["cluster-2"] = &storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
		ClusterName:     "test_cluster-2",
		ClusterId:       "test_cluster-2-id",
		Reason:          "timeout",
		OperatorVersion: "v1.6.0",
	}
	results := make(map[string][]*report.ResultRow)
	results["cluster-2"] = []*report.ResultRow{}
	return results, failedClusters
}

func getFakeEmptyFailedClusters() map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster {
	return make(map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster)
}

type emptyValueMatcher struct {
	value string
	data  map[string][]*report.ResultRow
	error string
}

func (m *emptyValueMatcher) Matches(target interface{}) bool {
	record, ok := target.(csv.Value)
	if !ok {
		m.error = "target is not of type csv.Value"
		return false
	}
	for range m.data {
		m.error = fmt.Sprintf("expected record: %s", m.value)
		return compareStringSlice(record, []string{m.value})
	}
	return false
}

func (m *emptyValueMatcher) String() string {
	return m.error
}

type valueMatcher struct {
	recordNumber int
	data         map[string][]*report.ResultRow
	error        string
}

func (m *valueMatcher) Matches(target interface{}) bool {
	record, ok := target.(csv.Value)
	if !ok {
		m.error = "target is not of type csv.Value"
		return false
	}
	recordIt := 0
	for _, clusterData := range m.data {
		for _, check := range clusterData {
			if recordIt == m.recordNumber {
				m.error = fmt.Sprintf("expected record: %v", generateRecord(check))
				return compareStringSlice(record, generateRecord(check))
			}
			recordIt++
		}
	}
	return false
}

func compareStringSlice(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (m *valueMatcher) String() string {
	return m.error
}

type failedClusterValueMatcher struct {
	data   map[string][]*report.ResultRow
	failed map[string]*storage.ComplianceOperatorReportSnapshotV2_FailedCluster
	error  string
}

func (m *failedClusterValueMatcher) Matches(target interface{}) bool {
	record, ok := target.(csv.Value)
	if !ok {
		m.error = "target is not of type csv.Value"
		return false
	}
	for _, clusterData := range m.data {
		for _, check := range clusterData {
			if compareStringSlice(record, generateRecord(check)) {
				return true
			}
		}
	}

	for _, failedCluster := range m.failed {
		if compareStringSlice(record, generateFailRecord(failedCluster)) {
			return true
		}
	}

	m.error = fmt.Sprintf("AddValue called with an unexpected record %v", record)
	return false
}

func (m *failedClusterValueMatcher) String() string {
	return m.error
}
