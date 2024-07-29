/* eslint-disable no-nested-ternary */
import React, { useState } from 'react';
import {
    Alert,
    AlertActionCloseButton,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Card,
    CardBody,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import { complianceEnhancedSchedulesPath } from 'routePaths';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';
import useAlert from 'hooks/useAlert';
import useFeatureFlags from 'hooks/useFeatureFlags';
import {
    ComplianceScanConfigurationStatus,
    runComplianceReport,
    runComplianceScanConfiguration,
} from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    getBodyDefault,
    getSubjectDefault,
    getTimeWithHourMinuteFromISO8601,
} from './compliance.scanConfigs.utils';
import ScanConfigParametersView from './components/ScanConfigParametersView';
import ScanConfigProfilesView from './components/ScanConfigProfilesView';
import ScanConfigClustersTable from './components/ScanConfigClustersTable';

import ScanConfigActionDropdown from './ScanConfigActionDropdown';

const headingLevel = 'h2';

type ViewScanConfigDetailProps = {
    hasWriteAccessForCompliance: boolean;
    scanConfig?: ComplianceScanConfigurationStatus;
    isLoading: boolean;
    error?: Error | string | null;
};

function ViewScanConfigDetail({
    hasWriteAccessForCompliance,
    scanConfig,
    isLoading,
    error = null,
}: ViewScanConfigDetailProps): React.ReactElement {
    const [isTriggeringRescan, setIsTriggeringRescan] = useState(false);
    const { alertObj, setAlertObj, clearAlertObj } = useAlert();

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isComplianceReportingEnabled = isFeatureFlagEnabled('ROX_COMPLIANCE_REPORTING');

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
            <Divider component="div" />
            <PageSection isCenterAligned>
                {isLoading ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : (
                    error && (
                        <Alert
                            variant="warning"
                            title="Unable to fetch scan schedule"
                            component="p"
                            isInline
                        >
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    )
                )}
                {!isLoading && scanConfig && (
                    <Card>
                        <CardBody>
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsLg' }}
                            >
                                <ScanConfigParametersView
                                    headingLevel={headingLevel}
                                    scanName={scanConfig.scanName}
                                    description={scanConfig.scanConfig.description}
                                    scanSchedule={scanConfig.scanConfig.scanSchedule}
                                >
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Last run</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {scanConfig.lastExecutedTime
                                                ? getTimeWithHourMinuteFromISO8601(
                                                      scanConfig.lastExecutedTime
                                                  )
                                                : 'Scan is in progress'}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Last updated</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {getTimeWithHourMinuteFromISO8601(
                                                scanConfig.lastUpdatedTime
                                            )}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                </ScanConfigParametersView>
                                <ScanConfigClustersTable
                                    headingLevel={headingLevel}
                                    clusterScanStatuses={scanConfig.clusterStatus}
                                />
                                <ScanConfigProfilesView
                                    headingLevel={headingLevel}
                                    profiles={scanConfig.scanConfig.profiles}
                                />
                                {isComplianceReportingEnabled && (
                                    <NotifierConfigurationView
                                        customBodyDefault={getBodyDefault(
                                            scanConfig.scanConfig.profiles
                                        )}
                                        customSubjectDefault={getSubjectDefault(
                                            scanConfig.scanName,
                                            scanConfig.scanConfig.profiles
                                        )}
                                        notifierConfigurations={scanConfig.scanConfig.notifiers}
                                    />
                                )}
                            </Flex>
                        </CardBody>
                    </Card>
                )}
            </PageSection>
        </>
    );
}

export default ViewScanConfigDetail;
