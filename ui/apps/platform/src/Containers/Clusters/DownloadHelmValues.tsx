import React, { useState, ReactElement } from 'react';
import { connect } from 'react-redux';
import { SuccessButton } from '@stackrox/ui-components';

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
            titleClassName="border-b px-1 border-primary-300 leading-normal cursor-pointer flex justify-between items-center bg-primary-200 hover:border-primary-400"
        >
            <div className="w-full p-3 leading-normal border-b pb-3 border-primary-300">
                {description}
            </div>
            <div className="flex justify-center items-center p-4">
                <SuccessButton type="button" onClick={downloadValues} isDisabled={isFetchingValues}>
                    Download Helm values
                </SuccessButton>
            </div>
        </CollapsibleCard>
    );
};
const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(DownloadHelmValues);
