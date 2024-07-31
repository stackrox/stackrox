import React from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
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
import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ScanConfigWizardForm from './Wizard/ScanConfigWizardForm';
import { defaultScanConfigFormValues } from './Wizard/useFormikScanConfig';
import { convertScanConfigToFormik } from './compliance.scanConfigs.utils';

type EditScanConfigDetailProps = {
    scanConfig?: ComplianceScanConfigurationStatus;
    isLoading: boolean;
    error?: Error | string | null;
};

function EditScanConfigDetail({
    scanConfig,
    isLoading,
    error = null,
}: EditScanConfigDetailProps): React.ReactElement {
    const parsedScanConfig = scanConfig
        ? convertScanConfigToFormik(scanConfig)
        : defaultScanConfigFormValues;

    return (
        <>
            <PageTitle title="Edit Compliance Scan Schedule Details" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedSchedulesPath}>
                        Scan schedules
                    </BreadcrumbItemLink>
                    {!isLoading && !error && scanConfig && (
                        <BreadcrumbItem isActive>Edit {scanConfig.scanName}</BreadcrumbItem>
                    )}
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                {!isLoading && !error && scanConfig && (
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        className="pf-v5-u-py-lg pf-v5-u-px-lg"
                    >
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <Title headingLevel="h1">Edit {scanConfig.scanName}</Title>
                        </FlexItem>
                    </Flex>
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
                    <PageSection padding={{ default: 'noPadding' }} isCenterAligned>
                        <ScanConfigWizardForm initialFormValues={parsedScanConfig} />
                    </PageSection>
                )}
            </PageSection>
        </>
    );
}

export default EditScanConfigDetail;
