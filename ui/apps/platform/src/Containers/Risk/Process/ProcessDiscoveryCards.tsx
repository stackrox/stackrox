import React from 'react';
import orderBy from 'lodash/orderBy';

import type { ProcessNameAndContainerNameGroup } from 'services/ProcessService';

import ProcessDiscoveryCard from './ProcessDiscoveryCard';

export type ProcessDiscoveryCardsProps = {
    deploymentId: string;
    processGroups: ProcessNameAndContainerNameGroup[];
    processEpoch: number;
    setProcessEpoch: (number) => void;
};

function ProcessDiscoveryCards({
    deploymentId,
    processGroups,
    processEpoch,
    setProcessEpoch,
}: ProcessDiscoveryCardsProps) {
    const sortedProcessGroups = orderBy(processGroups, ['suspicious', 'name'], ['desc', 'asc']);
    return sortedProcessGroups.map((pg, i, list) => (
        <div
            className={`px-3 ${i === list.length - 1 ? '' : 'pb-5'}`}
            key={pg.name}
            data-testid="process-discovery-card"
        >
            <ProcessDiscoveryCard
                process={pg}
                deploymentId={deploymentId}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
        </div>
    ));
}

export default ProcessDiscoveryCards;
