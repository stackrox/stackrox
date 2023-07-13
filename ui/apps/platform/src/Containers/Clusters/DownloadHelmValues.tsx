import React, { useState, ReactElement } from 'react';
import { connect } from 'react-redux';
import { Button } from '@patternfly/react-core';
import { DownloadIcon } from '@patternfly/react-icons';

import CollapsibleCard from 'Components/CollapsibleCard';
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

    return (
        <CollapsibleCard
            cardClassName="flex-grow border border-base-400 md:self-start"
            isCollapsible={false}
            title="Download helm values"
        >
            <div className="w-full p-3 leading-normal border-b pb-3 border-primary-300">
                {description}
            </div>
            <div className="flex justify-center items-center p-4">
                <Button
                    variant="secondary"
                    icon={<DownloadIcon />}
                    onClick={downloadValues}
                    isDisabled={isFetchingValues}
                    isLoading={isFetchingValues}
                >
                    Download Helm values
                </Button>
            </div>
        </CollapsibleCard>
    );
};
const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(DownloadHelmValues);
