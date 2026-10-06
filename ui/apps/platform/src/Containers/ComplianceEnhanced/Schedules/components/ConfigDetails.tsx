import { Alert, Bullseye, Flex, Spinner } from '@patternfly/react-core';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ScanConfigActivityView from './ScanConfigActivityView';
import ScanConfigClustersTable from './ScanConfigClustersTable';
import ScanConfigDeliveryView from './ScanConfigDeliveryView';
import ScanConfigParametersView from './ScanConfigParametersView';
import ScanConfigProfilesView from './ScanConfigProfilesView';

export type ConfigDetailsProps = {
    scanConfig?: ComplianceScanConfigurationStatus;
    isLoading?: boolean;
    error?: Error | string | null;
};

const headingLevel = 'h2';

function ConfigDetails({ isLoading, error, scanConfig }: ConfigDetailsProps) {
    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Alert variant="warning" title="Unable to fetch scan schedule" component="p" isInline>
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    if (scanConfig) {
        return (
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                <ScanConfigParametersView
                    headingLevel={headingLevel}
                    scanName={scanConfig.scanName}
                    description={scanConfig.scanConfig.description}
                    scanSchedule={scanConfig.scanConfig.scanSchedule}
                />
                <ScanConfigActivityView headingLevel={headingLevel} scanConfig={scanConfig} />
                <ScanConfigClustersTable
                    headingLevel={headingLevel}
                    clusterScanStatuses={scanConfig.clusterStatus}
                />
                <ScanConfigProfilesView
                    headingLevel={headingLevel}
                    profiles={scanConfig.scanConfig.profiles}
                />
                <ScanConfigDeliveryView
                    headingLevel={headingLevel}
                    notifiers={scanConfig.scanConfig.notifiers}
                />
            </Flex>
        );
    }
}

export default ConfigDetails;
