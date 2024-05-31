/* eslint-disable no-nested-ternary */
import React, { useState } from 'react';
import { generatePath } from 'react-router-dom';
import {
    Alert,
    AlertActionCloseButton,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Button,
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
    Timestamp,
    Title,
} from '@patternfly/react-core';

import { complianceEnhancedSchedulesPath } from 'routePaths';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import LinkShim from 'Components/PatternFly/LinkShim';
import NotifierConfigurationView from 'Components/NotifierConfiguration/NotifierConfigurationView';
import useAlert from 'hooks/useAlert';
import useFeatureFlags from 'hooks/useFeatureFlags';
import {
    runComplianceScanConfiguration,
    ComplianceScanConfigurationStatus,
} from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { scanConfigDetailsPath } from './compliance.scanConfigs.routes';
import { customBodyDefault, getSubjectDefault } from './compliance.scanConfigs.utils';
import ScanConfigParametersView from './components/ScanConfigParametersView';
import ScanConfigProfilesView from './components/ScanConfigProfilesView';
import ScanConfigClustersTable from './components/ScanConfigClustersTable';

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

    function onTriggerRescan() {
        if (scanConfig?.id) {
            clearAlertObj();
            setIsTriggeringRescan(true);

            runComplianceScanConfiguration(scanConfig.id)
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
                                    <Button
                                        variant="secondary"
                                        component={Button}
                                        onClick={onTriggerRescan}
                                        isLoading={isTriggeringRescan}
                                        isDisabled={isTriggeringRescan}
                                    >
                                        Re-scan
                                    </Button>
                                </FlexItem>
                            )}
                            {hasWriteAccessForCompliance && (
                                <FlexItem align={{ default: 'alignRight' }}>
                                    <Button
                                        variant="primary"
                                        component={LinkShim}
                                        href={`${generatePath(scanConfigDetailsPath, {
                                            scanConfigId: scanConfig.id,
                                        })}?action=edit`}
                                        isDisabled={!scanConfig || isTriggeringRescan}
                                    >
                                        Edit scan schedule
                                    </Button>
                                </FlexItem>
                            )}
                        </Flex>
                        {alertObj !== null && (
                            <Alert
                                title={alertObj.title}
                                variant={alertObj.type}
                                className="pf-v5-u-mb-lg pf-v5-u-mx-lg"
                                component="h2"
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
                            component="div"
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
                                            {scanConfig.lastExecutedTime ? (
                                                <Timestamp
                                                    date={new Date(scanConfig.lastExecutedTime)}
                                                    dateFormat="short"
                                                    timeFormat="long"
                                                    className="pf-v5-u-color-100 pf-v5-u-font-size-md"
                                                />
                                            ) : (
                                                'Scan is in progress'
                                            )}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Last updated</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Timestamp
                                                date={new Date(scanConfig.lastUpdatedTime)}
                                                dateFormat="short"
                                                timeFormat="long"
                                                className="pf-v5-u-color-100 pf-v5-u-font-size-md"
                                            />
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
                                        customBodyDefault={customBodyDefault}
                                        customSubjectDefault={getSubjectDefault(
                                            scanConfig.scanName
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
