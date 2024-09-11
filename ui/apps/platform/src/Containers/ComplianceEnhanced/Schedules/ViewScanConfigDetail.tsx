/* eslint-disable no-nested-ternary */
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

import { complianceEnhancedSchedulesPath } from 'routePaths';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import useAlert from 'hooks/useAlert';
import {
    ComplianceScanConfigurationStatus,
    runComplianceReport,
    runComplianceScanConfiguration,
} from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { JobContextTab } from 'types/reportJob';
import { ensureJobContextTab } from 'utils/reportJob';
import ScanConfigActionDropdown from './ScanConfigActionDropdown';
import ConfigDetails from './components/ConfigDetails';
import ReportJobs from './components/ReportJobs';
import ReportJobsHelpAction from '../../../Components/ReportJobsHelpAction';

type ViewScanConfigDetailProps = {
    hasWriteAccessForCompliance: boolean;
    scanConfig?: ComplianceScanConfigurationStatus;
    isLoading: boolean;
    error?: Error | string | null;
};

const configDetailsTabId = 'ComplianceScanConfigDetails';
const allReportJobsTabId = 'ComplianceScanConfigReportJobs';

function ViewScanConfigDetail({
    hasWriteAccessForCompliance,
    scanConfig,
    isLoading,
    error = null,
}: ViewScanConfigDetailProps): React.ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isReportJobsEnabled = isFeatureFlagEnabled('ROX_SCAN_SCHEDULE_REPORT_JOBS');

    const [selectedTab, setSelectedTab] = useState<JobContextTab>('CONFIGURATION_DETAILS');
    const [isTriggeringRescan, setIsTriggeringRescan] = useState(false);

    const { alertObj, setAlertObj, clearAlertObj } = useAlert();

    function handleRunScanConfig(scanConfigResponse: ComplianceScanConfigurationStatus) {
        clearAlertObj();
        setIsTriggeringRescan(true);

        runComplianceScanConfiguration(scanConfigResponse.id)
            .then(() => {
                setAlertObj({
                    type: 'success',
                    title: 'Successfully triggered a re-scan',
                });
                // TODO verify is lastExecutedTime expected to change? therefore, refetch?
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
        runComplianceReport(scanConfigResponse.id)
            .then(() => {
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
                                        isScanning={
                                            isTriggeringRescan /* ||
                                            scanConfig.lastExecutedTime === null */
                                        }
                                        scanConfigResponse={scanConfig}
                                    />
                                </FlexItem>
                            )}
                        </Flex>
                        {alertObj !== null && (
                            <Alert
                                title={alertObj.title}
                                component="p"
                                variant={alertObj.type}
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
                        activeKey={selectedTab}
                        onSelect={(_e, tab) => {
                            setSelectedTab(ensureJobContextTab(tab));
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
            {selectedTab === 'CONFIGURATION_DETAILS' && (
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
            {selectedTab === 'ALL_REPORT_JOBS' && (
                <PageSection isCenterAligned id={allReportJobsTabId}>
                    <ReportJobs scanConfig={scanConfig} />
                </PageSection>
            )}
        </>
    );
}

export default ViewScanConfigDetail;
