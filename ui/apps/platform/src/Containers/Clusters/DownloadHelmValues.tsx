import React, { useState, ReactElement } from 'react';
import { SuccessButton } from '@stackrox/ui-components';

import CollapsibleCard from 'Components/CollapsibleCard';
import { downloadClusterHelmValuesYaml } from 'services/ClustersService';

export type DownloadHelmValuesProps = {
    clusterId: string;
};

const DownloadHelmValues = ({ clusterId }: DownloadHelmValuesProps): ReactElement => {
    const [isFetchingValues, setIsFetchingValues] = useState(false);

    function downloadValues(): void {
        setIsFetchingValues(true);
        downloadClusterHelmValuesYaml(clusterId)
            .catch(() => {
                // TODO display message when there is a place for minor errors
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
                Download the required YAML to update your Helm values.
            </div>
            <div className="flex justify-center items-center p-4">
                <SuccessButton type="button" onClick={downloadValues} isDisabled={isFetchingValues}>
                    Download Helm values
                </SuccessButton>
            </div>
        </CollapsibleCard>
    );
};

export default DownloadHelmValues;
