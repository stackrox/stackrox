import React, { useState, ReactElement } from 'react';
import { connect } from 'react-redux';
import { Button, Flex, FlexItem, Text, Title } from '@patternfly/react-core';
import { DownloadIcon } from '@patternfly/react-icons';

import useAnalytics, { LEGACY_CLUSTER_DOWNLOAD_HELM_VALUES } from 'hooks/useAnalytics';
import { actions as notificationActions } from 'reducers/notifications';
import { downloadClusterHelmValuesYaml } from 'services/ClustersService';

export type DownloadHelmValuesProps = {
    clusterId: string;
    description: string;
    addToast: (message: string) => void;
    removeToast: () => void;
};

const DownloadHelmValues = ({
    clusterId,
    description,
    addToast,
    removeToast,
}: DownloadHelmValuesProps): ReactElement => {
    const { analyticsTrack } = useAnalytics();
    const [isFetchingValues, setIsFetchingValues] = useState(false);

    function downloadValues(): void {
        setIsFetchingValues(true);
        downloadClusterHelmValuesYaml(clusterId)
            .catch((err: { message: string }) => {
                addToast(`Problem downloading the Helm values. Please try again. (${err.message})`);
                setTimeout(removeToast, 5000);
            })
            .finally(() => {
                setIsFetchingValues(false);
            });
    }

    // Without FlexItem element, Button stretches to column width.
    return (
        <Flex direction={{ default: 'column' }}>
            <Title headingLevel="h2">Download helm values</Title>
            <Text>{description}</Text>
            <FlexItem>
                <Button
                    variant="secondary"
                    icon={<DownloadIcon />}
                    onClick={() => {
                        downloadValues();
                        analyticsTrack(LEGACY_CLUSTER_DOWNLOAD_HELM_VALUES);
                    }}
                    isDisabled={isFetchingValues}
                    isLoading={isFetchingValues}
                >
                    Download Helm values
                </Button>
            </FlexItem>
        </Flex>
    );
};
const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(DownloadHelmValues);
