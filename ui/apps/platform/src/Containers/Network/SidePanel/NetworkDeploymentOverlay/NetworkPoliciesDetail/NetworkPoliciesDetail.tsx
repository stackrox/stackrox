import React, { ReactElement, useEffect, useState } from 'react';
import { Download } from 'react-feather';

import Button from 'Components/Button';
import CollapsibleCard from 'Components/CollapsibleCard';
import Loader from 'Components/Loader';
import NoResultsMessage from 'Components/NoResultsMessage';
import { fetchNetworkPolicies } from 'services/NetworkService';
import download from 'utils/download';

// TODO delete type definition when it is available to import.
type NetworkPolicy = {
    id: string;
    name: string;
    yaml: string;
};

export type NetworkPoliciesDetailProps = {
    policyIds: string[];
};

function downloadYamlFile(name: string, content: string, type: string) {
    return (): void => download(name, content, type);
}

function NetworkPoliciesDetail({ policyIds }: NetworkPoliciesDetailProps): ReactElement {
    const [isLoading, setIsLoading] = useState(false);
    const [networkPolicies, setNetworkPolicies] = useState<NetworkPolicy[]>([]);

    useEffect(() => {
        setIsLoading(true);
        fetchNetworkPolicies(policyIds)
            .then(
                (allResponses) => {
                    setNetworkPolicies(allResponses?.response || []);
                },
                () => setNetworkPolicies([])
            )
            .finally(() => {
                setIsLoading(false);
            });
    }, [policyIds, setNetworkPolicies]);

    return (
        <div className="flex flex-col bg-base-100 rounded border border-base-400 overflow-y-auto p-3 w-full h-full">
            {isLoading && <Loader />}
            {!isLoading && networkPolicies.length < 1 && (
                <NoResultsMessage
                    message="No network policies defined for this deployment"
                    className="text-lg"
                    icon="warn"
                />
            )}
            {networkPolicies.map((networkPolicy) => {
                const { id, name, yaml } = networkPolicy;
                // text-base restores 12px font size which text-sm overrides in div element for overlay ancestor
                return (
                    <CollapsibleCard title={name} cardClassName="border border-base-400" key={id}>
                        <div className="p-4 bg-primary-100">
                            <pre className="font-600 h-full leading-normal p-3 text-base whitespace-pre-wrap word-break">
                                {yaml}
                            </pre>
                        </div>
                        <div className="flex justify-center p-3 border-t border-base-400">
                            <Button
                                className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300 hover:bg-primary-100"
                                onClick={downloadYamlFile(`${name}.yaml`, yaml, 'yaml')}
                                tabIndex="-1"
                                icon={<Download className="h-3 w-3 mr-4" />}
                                text="Download YAML file"
                            />
                        </div>
                    </CollapsibleCard>
                );
            })}
        </div>
    );
}

export default NetworkPoliciesDetail;
