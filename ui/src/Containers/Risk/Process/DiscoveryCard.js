import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import CollapsibleCard from 'Components/CollapsibleCard';

import ProcessesDiscoveryCardHeader from './DiscoveryCardHeader';

function ProcessesDiscoveryCard({
    deploymentId,
    process,
    processEpoch,
    setProcessEpoch,
    children,
}) {
    function renderWhenOpened() {
        return (
            <ProcessesDiscoveryCardHeader
                icon={<Icon.ChevronUp className="h-4 w-4" />}
                deploymentId={deploymentId}
                process={process}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
        );
    }

    function renderWhenClosed() {
        return (
            <ProcessesDiscoveryCardHeader
                icon={<Icon.ChevronDown className="h-4 w-4" />}
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
            {children}
        </CollapsibleCard>
    );
}

ProcessesDiscoveryCard.propTypes = {
    deploymentId: PropTypes.string.isRequired,
    process: PropTypes.shape({
        name: PropTypes.string.isRequired,
        containerName: PropTypes.string.isRequired,
        suspicious: PropTypes.bool.isRequired,
        groups: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    }).isRequired,
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired,
    children: PropTypes.node.isRequired,
};

export default ProcessesDiscoveryCard;
