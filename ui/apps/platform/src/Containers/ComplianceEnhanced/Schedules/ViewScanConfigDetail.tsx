import React, { useState } from 'react';
import {
    Alert,
    AlertActionCloseButton,
    Breadcrumb,
    BreadcrumbItem,
    Card,
    CardBody,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Tab,
    Tabs,
    TabTitleText,
    Title,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import { complianceEnhancedSchedulesPath } from 'routePaths';
import useAlert from 'hooks/useAlert';
import useURLStringUnion from 'hooks/useURLStringUnion';
import {
    ComplianceScanConfigurationStatus,
    runComplianceReport,
    runComplianceScanConfiguration,
} from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ReportJobsHelpAction from 'Components/ReportJob/ReportJobsHelpAction';
import { jobContextTabs } from 'Components/ReportJob/types';
import useAnalytics from 'hooks/useAnalytics';
import ScanConfigActionDropdown from './ScanConfigActionDropdown';
import ConfigDetails from './components/ConfigDetails';
import ReportJobs from './components/ReportJobs';
import useWatchLastSnapshotForComplianceReports from './hooks/useWatchLastSnapshotForComplianceReports';

type ViewScanConfigDetailProps = {
    hasWriteAccessForCompliance: boolean;
    isReportJobsEnabled: boolean;
    scanConfig?: ComplianceScanConfigurationStatus;
    isLoading: boolean;
    error?: Error | string | null;
};

const configDetailsTabId = 'ComplianceScanConfigDetails';
const allReportJobsTabId = 'ComplianceScanConfigReportJobs';

function ViewScanConfigDetail({
    hasWriteAccessForCompliance,
    isReportJobsEnabled,
    scanConfig,
    isLoading,
    error = null,
}: ViewScanConfigDetailProps): React.ReactElement {
    const { scanConfigId } = useParams();
    const { analyticsTrack } = useAnalytics();

    const [activeScanConfigTab, setActiveScanConfigTab] = useURLStringUnion(
        'scanConfigTab',
        jobContextTabs
    );
    const [isTriggeringRescan, setIsTriggeringRescan] = useState(false);

    const { alertObj, setAlertObj, clearAlertObj } = useAlert();
    const { complianceReportSnapshots } = useWatchLastSnapshotForComplianceReports(scanConfig);
    const lastSnapshot = complianceReportSnapshots[scanConfigId];

    const isReportStatusPending =
        lastSnapshot?.reportStatus.runState === 'PREPARING' ||
        lastSnapshot?.reportStatus.runState === 'WAITING';

    function handleRunScanConfig(scanConfigResponse: ComplianceScanConfigurationStatus) {
        clearAlertObj();
        setIsTriggeringRescan(true);

        runComplianceScanConfiguration(scanConfigResponse.id)
            .then(() => {
                setAlertObj({
                    type: 'success',
                    title: 'Successfully triggered a re-scan',
                });
            })
            .catch((error) => {
                setAlertObj({
                    type: 'danger',
                    title: 'Could not trigger a re-scan',
                    children: getAxiosErrorMessage(error),
                });
            })
            .finally(() => {
                setIsTriggeringRescan(false);
            });
    }

    function handleSendReport(scanConfigResponse: ComplianceScanConfigurationStatus) {
        clearAlertObj();
        runComplianceReport(scanConfigResponse.id, 'EMAIL')
            .then(() => {
                analyticsTrack({
                    event: 'Compliance Report Manual Send Triggered',
                    properties: { source: 'Details page' },
                });
                setAlertObj({
                    type: 'success',
                    title: 'Successfully requested to send a report',
                });
            })
            .catch((error) => {
                setAlertObj({
                    type: 'danger',
                    title: 'Could not send a report',
                    children: getAxiosErrorMessage(error),
                });
            });
    }

    function handleGenerateDownload(scanConfigResponse: ComplianceScanConfigurationStatus) {
        clearAlertObj();
        runComplianceReport(scanConfigResponse.id, 'DOWNLOAD')
            .then(() => {
                analyticsTrack({
                    event: 'Compliance Report Download Generation Triggered',
                    properties: { source: 'Details page' },
                });
                setAlertObj({
                    type: 'success',
                    title: 'The report generation has started and will be available for download once complete',
                });
            })
            .catch((error) => {
                setAlertObj({
                    type: 'danger',
                    title: 'Could not send a report',
                    children: getAxiosErrorMessage(error),
                });
            });
    }

    return (
        <>
            <PageTitle title="Compliance Scan Schedule Details" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedSchedulesPath}>
                        Scan schedules
                    </BreadcrumbItemLink>
                    {!isLoading && !error && scanConfig && (
                        <BreadcrumbItem isActive>{scanConfig.scanName}</BreadcrumbItem>
                    )}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                {!isLoading && !error && scanConfig && (
                    <>
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            className="pf-v5-u-py-lg pf-v5-u-px-lg"
                        >
                            <FlexItem flex={{ default: 'flex_1' }}>
                                <Title headingLevel="h1">{scanConfig.scanName}</Title>
                            </FlexItem>
                            {hasWriteAccessForCompliance && (
                                <FlexItem align={{ default: 'alignRight' }}>
                                    <ScanConfigActionDropdown
                                        handleRunScanConfig={handleRunScanConfig}
                                        handleSendReport={handleSendReport}
                                        handleGenerateDownload={handleGenerateDownload}
                                        isScanning={isTriggeringRescan}
                                        isReportStatusPending={isReportStatusPending}
                                        scanConfigResponse={scanConfig}
                                        isReportJobsEnabled={isReportJobsEnabled}
                                    />
                                </FlexItem>
                            )}
                        </Flex>
                        {alertObj !== null && (
                            <Alert
                                title={alertObj.title}
                                component="p"
                                variant={alertObj.type}
                                isInline
                                className="pf-v5-u-mb-lg pf-v5-u-mx-lg"
                                actionClose={<AlertActionCloseButton onClose={clearAlertObj} />}
                            >
                                {alertObj.children}
                            </Alert>
                        )}
                    </>
                )}
            </PageSection>
            {isReportJobsEnabled && (
                <PageSection variant="light" className="pf-v5-u-py-0">
                    <Tabs
                        activeKey={activeScanConfigTab}
                        onSelect={(_e, tab) => {
                            setActiveScanConfigTab(tab);
                            if (tab === 'ALL_REPORT_JOBS') {
                                analyticsTrack('Compliance Report Jobs Table Viewed');
                            }
                        }}
                        aria-label="Scan schedule details tabs"
                    >
                        <Tab
                            tabContentId={configDetailsTabId}
                            eventKey="CONFIGURATION_DETAILS"
                            title={<TabTitleText>Configuration details</TabTitleText>}
                        />
                        <Tab
                            tabContentId={allReportJobsTabId}
                            eventKey="ALL_REPORT_JOBS"
                            title={<TabTitleText>All report jobs</TabTitleText>}
                            actions={<ReportJobsHelpAction reportType="Scan schedule" />}
                        />
                    </Tabs>
                </PageSection>
            )}
            {activeScanConfigTab === 'CONFIGURATION_DETAILS' && (
                <PageSection isCenterAligned id={configDetailsTabId}>
                    <Card isFlat>
                        <CardBody>
                            <ConfigDetails
                                isLoading={isLoading}
                                error={error}
                                scanConfig={scanConfig}
                            />
                        </CardBody>
                    </Card>
                </PageSection>
            )}
            {activeScanConfigTab === 'ALL_REPORT_JOBS' && scanConfig?.id && (
                <PageSection isCenterAligned id={allReportJobsTabId}>
                    <ReportJobs scanConfigId={scanConfig.id} />
                </PageSection>
            )}
        </>
    );
}

export default ViewScanConfigDetail;
