import type { ReactElement } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import { complianceEnhancedSchedulesPath } from 'routePaths';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ScanConfigWizardPageSection from './Wizard/ScanConfigWizardPageSection';
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

    return (
        <>
            <PageTitle title="Edit Compliance Scan Schedule" />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={complianceEnhancedSchedulesPath}>
                        Scan schedules
                    </BreadcrumbItemLink>
                    {scanConfig && (
                        <BreadcrumbItem isActive>Edit {scanConfig.scanName}</BreadcrumbItem>
                    )}
                </Breadcrumb>
            </PageSection>
            {isLoading ? (
                <PageSection isFilled>
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                </PageSection>
            ) : (
                <>
                    {scanConfig && (
                        <PageSection>
                            <Title headingLevel="h1">Edit {scanConfig.scanName}</Title>
                        </PageSection>
                    )}
                    {error && (
                        <PageSection>
                            <Alert
                                variant="warning"
                                title="Unable to fetch scan schedule"
                                component="p"
                                isInline
                            >
                                {getAxiosErrorMessage(error)}
                            </Alert>
                        </PageSection>
                    )}
                    {scanConfig && (
                        <ScanConfigWizardPageSection
                            initialFormValues={parsedScanConfig}
                            pageAction="edit"
                        />
                    )}
                </>
            )}
        </>
    );
}

export default EditScanConfigDetail;
