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
    Divider,
    Grid,
    GridItem,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import {
    complianceEnhancedScanConfigsPath,
    complianceEnhancedScanConfigDetailPath,
} from 'routePaths';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import LinkShim from 'Components/PatternFly/LinkShim';
import useAlert from 'hooks/useAlert';
import {
    runComplianceScanConfiguration,
    ComplianceScanConfigurationStatus,
} from 'services/ComplianceEnhancedService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ScanConfigParameterView from './components/ScanConfigParameterView';
import ScanConfigProfiles from './components/ScanConfigProfiles';
import ScanConfigClustersTable from './components/ScanConfigClustersTable';

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
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedScanConfigsPath}>
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
                            className="pf-u-py-lg pf-u-px-lg"
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
                                        isDisabled={!scanConfig || isTriggeringRescan}
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
                                        href={`${generatePath(
                                            complianceEnhancedScanConfigDetailPath,
                                            {
                                                scanConfigId: scanConfig.id,
                                            }
                                        )}?action=edit`}
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
                                className="pf-u-mb-lg pf-u-mx-lg"
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
                        <Spinner isSVG />
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
                    <Grid hasGutter>
                        <GridItem sm={12} md={6}>
                            <ScanConfigParameterView scanConfig={scanConfig} />
                        </GridItem>
                        <GridItem sm={12} md={6}>
                            <ScanConfigProfiles profiles={scanConfig.scanConfig.profiles} />
                        </GridItem>
                        <GridItem sm={12}>
                            <ScanConfigClustersTable
                                clusterScanStatuses={scanConfig.clusterStatus}
                            />
                        </GridItem>
                    </Grid>
                )}
            </PageSection>
        </>
    );
}

export default ViewScanConfigDetail;
