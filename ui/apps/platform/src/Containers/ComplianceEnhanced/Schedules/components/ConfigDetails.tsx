import { Alert, Bullseye, Flex, List, ListItem, Spinner } from '@patternfly/react-core';

import type { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import { getClusterScanErrors, hasScanConfigErrors } from '../compliance.scanConfigs.utils';
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
    const scanErrorItems = scanConfig?.clusterStatus.flatMap((clusterStatus) =>
        getClusterScanErrors(clusterStatus).map((scanError) => ({
            clusterId: clusterStatus.clusterId,
            clusterName: clusterStatus.clusterName || clusterStatus.clusterId,
            scanError,
        }))
    );

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
                {hasScanConfigErrors(scanConfig) && (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsXs' }}
                    >
                        <Alert variant="danger" title="Scan error" component="p" isInline>
                            One or more clusters reported scan errors.
                        </Alert>
                        {scanErrorItems && scanErrorItems.length > 0 && (
                            <List>
                                {scanErrorItems.map(({ clusterId, clusterName, scanError }) => (
                                    <ListItem key={`${clusterId}-${scanError}`}>
                                        {clusterName}: {scanError}
                                    </ListItem>
                                ))}
                            </List>
                        )}
                    </Flex>
                )}
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
