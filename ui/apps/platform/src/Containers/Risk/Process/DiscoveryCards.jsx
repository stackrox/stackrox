import React from 'react';
import PropTypes from 'prop-types';
import orderBy from 'lodash/orderBy';

import ProcessDiscoveryCard from './DiscoveryCard';
import Binaries from './Binaries';

function DiscoveryCards({ deploymentId, processGroup, processEpoch, setProcessEpoch }) {
    const sortedProcessGroups = orderBy(
        processGroup.groups,
        ['suspicious', 'name'],
        ['desc', 'asc']
    );
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
            >
                <Binaries processes={pg.groups} />
            </ProcessDiscoveryCard>
        </div>
    ));
}

DiscoveryCards.propTypes = {
    deploymentId: PropTypes.string.isRequired,
    processGroup: PropTypes.shape({
        groups: PropTypes.arrayOf(PropTypes.object),
    }).isRequired,
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired,
};

export default DiscoveryCards;
