import CollapsibleCard from 'Components/CollapsibleCard';
import type { ProcessNameAndContainerNameGroup } from 'services/ProcessService';

import Binaries from './Binaries';
import ProcessDiscoveryCardHeader from './ProcessDiscoveryCardHeader';

export type ProcessDiscoveryCardProps = {
    deploymentId: string;
    process: ProcessNameAndContainerNameGroup;
    processEpoch: number;
    setProcessEpoch: (number) => void;
};

function ProcessDiscoveryCard({
    deploymentId,
    process,
    processEpoch,
    setProcessEpoch,
}: ProcessDiscoveryCardProps) {
    function renderWhenOpened() {
        return (
            <ProcessDiscoveryCardHeader
                isExpanded
                deploymentId={deploymentId}
                process={process}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
        );
    }

    function renderWhenClosed() {
        return (
            <ProcessDiscoveryCardHeader
                isExpanded={false}
                deploymentId={deploymentId}
                process={process}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
        );
    }

    return (
        <CollapsibleCard
            title={process.name}
            open={false}
            renderWhenOpened={renderWhenOpened}
            renderWhenClosed={renderWhenClosed}
            cardClassName="border border-base-400"
        >
            <Binaries processes={process.groups} />
        </CollapsibleCard>
    );
}

export default ProcessDiscoveryCard;
