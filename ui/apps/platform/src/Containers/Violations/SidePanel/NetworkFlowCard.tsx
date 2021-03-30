import React, { ReactElement } from 'react';
import { format } from 'date-fns';

import KeyValue from 'Components/KeyValue';
import RuntimeViolationCollapsibleCard from 'Containers/Violations/RuntimeViolationCollapsibleCard';
import dateTimeFormat from 'constants/dateTimeFormat';

export type NetworkFlowInfo = {
    protocol: string;
    source: {
        name: string;
        entityType: string;
        deploymentNamespace: string;
        deploymentType: string;
        port: string | number;
    };
    destination: {
        name: string;
        entityType: string;
        deploymentNamespace: string;
        deploymentType: string;
        port: string | number;
    };
};
export type NetworkFlowCardProps = {
    networkFlowInfo: NetworkFlowInfo;
    message: string;
    time: string;
};

function NetworkFlowCard({ networkFlowInfo, message, time }: NetworkFlowCardProps): ReactElement {
    const networkFlowsDetails = (
        <>
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue
                    className="w-1/2"
                    label="Source Entity Type:"
                    value={networkFlowInfo.source.entityType}
                />
                <KeyValue
                    className="w-1/2"
                    label="Source Name:"
                    value={networkFlowInfo.source.name}
                />
            </div>
            {(!!networkFlowInfo?.source?.deploymentType ||
                !!networkFlowInfo?.source?.deploymentNamespace) && (
                <div className="flex flex-1 text-base-600 px-4 py-2">
                    {!!networkFlowInfo?.source?.deploymentType && (
                        <KeyValue
                            className="w-1/2"
                            label="Source Deployment Type:"
                            value={networkFlowInfo.source.deploymentType}
                        />
                    )}
                    {!!networkFlowInfo?.source?.deploymentNamespace && (
                        <KeyValue
                            className="w-1/2"
                            label="Source Namespace:"
                            value={networkFlowInfo.source.deploymentNamespace}
                        />
                    )}
                </div>
            )}
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue
                    className="w-1/2"
                    label="Destination Entity Type:"
                    value={networkFlowInfo.destination.entityType}
                />
                <KeyValue
                    className="w-1/2"
                    label="Destination Name:"
                    value={networkFlowInfo.destination.name}
                />
            </div>
            {(!!networkFlowInfo?.destination?.deploymentType ||
                !!networkFlowInfo?.destination?.deploymentNamespace) && (
                <div className="flex flex-1 text-base-600 px-4 py-2">
                    {!!networkFlowInfo?.destination?.deploymentType && (
                        <KeyValue
                            className="w-1/2"
                            label="Destination Deployment Type:"
                            value={networkFlowInfo.destination.deploymentType}
                        />
                    )}
                    {!!networkFlowInfo?.destination?.deploymentNamespace && (
                        <KeyValue
                            className="w-1/2"
                            label="Destination Namespace:"
                            value={networkFlowInfo.destination.deploymentNamespace}
                        />
                    )}
                </div>
            )}
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue
                    className="w-1/2"
                    label="Destination Port:"
                    value={networkFlowInfo.destination.port as string}
                />
                <KeyValue className="w-1/2" label="Protocol:" value={networkFlowInfo.protocol} />
            </div>
        </>
    );

    return (
        <div className="mb-4" key={message} data-testid="networkFlow">
            <RuntimeViolationCollapsibleCard title={message}>
                <div className="border-t border-base-300">
                    {networkFlowsDetails}
                    <div className="flex px-4 py-2 border-base-300 border-b text-base-600">
                        <KeyValue
                            label="Time:"
                            value={time ? format(time, dateTimeFormat) : 'N/A'}
                        />
                    </div>
                </div>
            </RuntimeViolationCollapsibleCard>
        </div>
    );
}

export default NetworkFlowCard;
