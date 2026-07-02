import type { ReactElement } from 'react';
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
import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
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
}: EditScanConfigDetailProps): ReactElement {
    const parsedScanConfig = scanConfig
        ? convertScanConfigToFormik(scanConfig)
        : defaultScanConfigFormValues;
    const isDiscovered = !scanConfig?.modifiedBy?.id;
    const pageTitle = isDiscovered
        ? 'Edit Compliance Scan Schedule Notifications'
        : 'Edit Compliance Scan Schedule Details';
    const heading = isDiscovered
        ? `Edit notifications for ${scanConfig?.scanName ?? ''}`
        : `Edit ${scanConfig?.scanName ?? ''}`;

    return (
        <>
            <PageTitle title={pageTitle} />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedSchedulesPath}>
                        Scan schedules
                    </BreadcrumbItemLink>
                    {!isLoading && !error && scanConfig && (
                        <BreadcrumbItem isActive>{heading}</BreadcrumbItem>
                    )}
                </Breadcrumb>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                {!isLoading && !error && scanConfig && (
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        className="pf-v6-u-py-lg pf-v6-u-px-lg"
                    >
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <Title headingLevel="h1">{heading}</Title>
                        </FlexItem>
                    </Flex>
                )}
            </PageSection>
            <Divider component="div" />
            <PageSection isFilled>
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
                    <PageSection padding={{ default: 'noPadding' }} isFilled>
                        <ScanConfigWizardForm
                            initialFormValues={parsedScanConfig}
                            isDiscovered={isDiscovered}
                        />
                    </PageSection>
                )}
            </PageSection>
        </>
    );
}

export default EditScanConfigDetail;
