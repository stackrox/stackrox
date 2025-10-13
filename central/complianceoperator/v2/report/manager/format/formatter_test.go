package format

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/central/complianceoperator/v2/report/manager/format/mocks"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	clusterID1 = "cluster-1"
	clusterID2 = "cluster-2"
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
	s.Run("with empty failed clusters", func() {
		timestamp := timestamppb.Now()
		clusterData := getFakeClusterData()
		fileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
		s.zipWriter.EXPECT().Create(fileName).Times(1).Return(nil, nil)
		gomock.InOrder(
			s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
				data := getFakeReportData()
				return compareStringSlice(s.T(), target, generateRecord(data[clusterID1][0]))
			})).Times(1),
			s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
				data := getFakeReportData()
				return compareStringSlice(s.T(), target, generateRecord(data[clusterID1][1]))
			})).Times(1),
		)
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(nil)
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(getFakeReportData(), getFakeClusterData())
		s.Require().NoError(err)
		s.Require().NotNil(buf)
	})
	s.Run("with nil clusters data", func() {
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)
		buf, err := s.formatter.FormatCSVReport(getFakeReportData(), nil)
		s.Require().NoError(err)
		s.Require().NotNil(buf)
	})
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportWithFailedClusterNoError() {
	timestamp := timestamppb.Now()
	results, clusterData := getFakeReportDataWithFailedCluster()

	successfulFileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
	failedFileName := getFileName(failedClusterFmt, clusterData[clusterID2].ClusterName, timestamp)

	calls := []any{
		s.zipWriter.EXPECT().Create(gomock.Cond[string](func(target string) bool {
			return target == successfulFileName
		})).Times(1).Return(nil, nil),
		s.zipWriter.EXPECT().Create(gomock.Cond[string](func(target string) bool {
			return target == failedFileName
		})).Times(1).Return(nil, nil),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			failedInfo := clusterData[clusterID2].FailedInfo
			return compareStringSlice(s.T(), target, []string{failedInfo.ClusterId, failedInfo.ClusterName, failedInfo.Reasons[0], failedInfo.OperatorVersion})
		})).Times(1),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			return compareStringSlice(s.T(), target, generateRecord(results[clusterID1][0]))
		})).Times(1),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			return compareStringSlice(s.T(), target, generateRecord(results[clusterID1][1]))
		})).Times(1),
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(2).Return(nil),
		s.zipWriter.EXPECT().Close().Times(1).Return(nil),
	}
	gomock.InAnyOrder(calls)

	buf, err := s.formatter.FormatCSVReport(getFakeReportDataWithFailedCluster())
	s.Require().NoError(err)
	s.Require().NotNil(buf)
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportWithFailedClusterInResultsParameterNoError() {
	timestamp := timestamppb.Now()
	results, clusterData := getFakeReportDataWithFailedCluster()
	// Add empty results to the failed cluster
	results[clusterID2] = []*report.ResultRow{}

	successfulFileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
	failedFileName := getFileName(failedClusterFmt, clusterData[clusterID2].ClusterName, timestamp)

	calls := []any{
		s.zipWriter.EXPECT().Create(gomock.Cond[string](func(target string) bool {
			return target == successfulFileName
		})).Times(1).Return(nil, nil),
		s.zipWriter.EXPECT().Create(gomock.Cond[string](func(target string) bool {
			return target == failedFileName
		})).Times(1).Return(nil, nil),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			failedInfo := clusterData[clusterID2].FailedInfo
			return compareStringSlice(s.T(), target, []string{failedInfo.ClusterId, failedInfo.ClusterName, failedInfo.Reasons[0], failedInfo.OperatorVersion})
		})).Times(1),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			return compareStringSlice(s.T(), target, generateRecord(results[clusterID1][0]))
		})).Times(1),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			return compareStringSlice(s.T(), target, generateRecord(results[clusterID1][1]))
		})).Times(1),
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(2).Return(nil),
		s.zipWriter.EXPECT().Close().Times(1).Return(nil),
	}
	gomock.InAnyOrder(calls)

	buf, err := s.formatter.FormatCSVReport(results, clusterData)
	s.Require().NoError(err)
	s.Require().NotNil(buf)
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportWithPartialFailedCluster() {
	timestamp := timestamppb.Now()
	results, clusterData := getFakeReportDataWithFailedCluster()
	// Add partial results
	results[clusterID2] = []*report.ResultRow{
		{
			ClusterName: "test_cluster-2",
			CheckName:   "test_check-2",
			Profile:     "test_profile-2",
			ControlRef:  "test_control_ref-2",
			Description: "description-2",
			Status:      "Fail",
			Remediation: "remediation-2",
		},
	}

	partialSuccessFileName := getFileName(successfulClusterFmt, clusterData[clusterID2].ClusterName, timestamp)
	successfulFileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
	failedFileName := getFileName(failedClusterFmt, clusterData[clusterID2].ClusterName, timestamp)

	calls := []any{
		s.zipWriter.EXPECT().Create(gomock.Cond[string](func(target string) bool {
			return target == successfulFileName
		})).Times(1).Return(nil, nil),
		s.zipWriter.EXPECT().Create(gomock.Cond[string](func(target string) bool {
			return target == failedFileName
		})).Times(1).Return(nil, nil),
		s.zipWriter.EXPECT().Create(gomock.Cond[string](func(target string) bool {
			return target == partialSuccessFileName
		})).Times(1).Return(nil, nil),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			failedInfo := clusterData[clusterID2].FailedInfo
			return compareStringSlice(s.T(), target, []string{failedInfo.ClusterId, failedInfo.ClusterName, failedInfo.Reasons[0], failedInfo.OperatorVersion})
		})).Times(1),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			return compareStringSlice(s.T(), target, generateRecord(results[clusterID1][0]))
		})).Times(1),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			return compareStringSlice(s.T(), target, generateRecord(results[clusterID1][1]))
		})).Times(1),
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			return compareStringSlice(s.T(), target, generateRecord(results[clusterID2][0]))
		})).Times(1),
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(3).Return(nil),
		s.zipWriter.EXPECT().Close().Times(1).Return(nil),
	}
	gomock.InAnyOrder(calls)

	buf, err := s.formatter.FormatCSVReport(results, clusterData)
	s.Require().NoError(err)
	s.Require().NotNil(buf)
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportCreateError() {
	s.Run("zip writer failing to create a file (with no failed clusters) should yield an error", func() {
		timestamp := timestamppb.Now()
		clusterData := getFakeClusterData()
		successfulFileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
		s.zipWriter.EXPECT().Create(successfulFileName).Times(1).Return(nil, errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(getFakeReportData(), getFakeClusterData())
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
	s.Run("zip writer failing to create a file (containing failed clusters) should yield an error", func() {
		timestamp := timestamppb.Now()
		results, clusterData := getFakeReportDataWithFailedCluster()
		delete(clusterData, clusterID1)
		failedFileName := getFileName(failedClusterFmt, clusterData[clusterID2].ClusterName, timestamp)
		s.zipWriter.EXPECT().Create(failedFileName).Times(1).Return(nil, errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(results, clusterData)
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportWriteError() {
	s.Run("csv writer failing to create a file (with no failed clusters) should yield an error", func() {
		timestamp := timestamppb.Now()
		clusterData := getFakeClusterData()
		successfulFileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
		s.zipWriter.EXPECT().Create(successfulFileName).Times(1).Return(nil, nil)
		s.csvWriter.EXPECT().AddValue(gomock.Any()).Times(2)
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(getFakeReportData(), getFakeClusterData())
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
	s.Run("csv writer failing to create a file (containing failed clusters) should yield an error", func() {
		timestamp := timestamppb.Now()
		results, clusterData := getFakeReportDataWithFailedCluster()
		delete(clusterData, clusterID1)
		failedFileName := getFileName(failedClusterFmt, clusterData[clusterID2].ClusterName, timestamp)
		s.zipWriter.EXPECT().Create(failedFileName).Times(1).Return(nil, nil)
		s.csvWriter.EXPECT().AddValue(gomock.Cond[csv.Value](func(target csv.Value) bool {
			failedInfo := clusterData[clusterID2].FailedInfo
			return compareStringSlice(s.T(), target, []string{failedInfo.ClusterId, failedInfo.ClusterName, failedInfo.Reasons[0], failedInfo.OperatorVersion})
		})).Times(1)
		s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(errors.New("error"))
		s.zipWriter.EXPECT().Close().Times(1).Return(nil)

		buf, err := s.formatter.FormatCSVReport(results, clusterData)
		s.Require().Error(err)
		s.Require().Nil(buf)
	})
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportCloseError() {
	timestamp := timestamppb.Now()
	clusterData := getFakeClusterData()
	fileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
	s.zipWriter.EXPECT().Create(fileName).Times(1).Return(nil, nil)
	s.csvWriter.EXPECT().AddValue(gomock.Any()).Times(2)
	s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(nil)
	s.zipWriter.EXPECT().Close().Times(1).Return(errors.New("error"))

	buf, err := s.formatter.FormatCSVReport(getFakeReportData(), clusterData)
	s.Require().Error(err)
	s.Require().Nil(buf)
}

func (s *ComplianceReportingFormatterSuite) Test_FormatCSVReportEmptyReportNoError() {
	timestamp := timestamppb.Now()
	clusterData := getFakeClusterData()
	fileName := getFileName(successfulClusterFmt, clusterData[clusterID1].ClusterName, timestamp)
	s.zipWriter.EXPECT().Create(fileName).Times(1).Return(nil, nil)
	s.csvWriter.EXPECT().AddValue(&emptyValueMatcher{
		t:     s.T(),
		value: emptyValue,
		data:  getFakeEmptyReportData(),
	}).Times(1)
	s.csvWriter.EXPECT().WriteCSV(gomock.Any()).Times(1).Return(nil)
	s.zipWriter.EXPECT().Close().Times(1).Return(nil)

	buf, err := s.formatter.FormatCSVReport(getFakeEmptyReportData(), getFakeClusterData())
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
	results[clusterID1] = []*report.ResultRow{
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

func getFakeReportDataWithFailedCluster() (map[string][]*report.ResultRow, map[string]*report.ClusterData) {
	clusterData := getFakeClusterData()
	clusterData[clusterID2] = &report.ClusterData{
		ClusterName: "test_cluster-2",
		ClusterId:   "test_cluster-2-id",
	}
	clusterData[clusterID2].FailedInfo = &report.FailedCluster{
		Reasons:         []string{"timeout"},
		OperatorVersion: "v1.6.0",
	}
	results := getFakeReportData()
	return results, clusterData
}

func getFakeClusterData() map[string]*report.ClusterData {
	ret := make(map[string]*report.ClusterData)
	ret[clusterID1] = &report.ClusterData{
		ClusterId:   clusterID1,
		ClusterName: "test_cluster-1",
	}
	return ret
}

type emptyValueMatcher struct {
	t     *testing.T
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
		return compareStringSlice(m.t, record, []string{m.value})
	}
	return false
}

func (m *emptyValueMatcher) String() string {
	return m.error
}

func compareStringSlice(t *testing.T, actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		t.Logf("Expected slice %v but got %v", expected, actual)
		return false
	}
	for i := 0; i < len(actual); i++ {
		if strings.Compare(actual[i], expected[i]) != 0 {
			t.Logf("Expected slice %v but got %v", expected, actual)
			return false
		}
	}
	return true
}
