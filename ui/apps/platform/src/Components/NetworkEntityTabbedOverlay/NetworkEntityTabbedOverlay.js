import React from 'react';
import PropTypes from 'prop-types';

import useTabs from 'hooks/useTabs';
import { networkEntityLabels } from 'messages/network';

import DetailsOverlay from 'Components/DetailsOverlay';
import NetworkEntityTabHeader from './NetworkEntityTabHeader';

function NetworkEntityTabbedOverlay({ entityName, entityType, children }) {
    const { tabHeaders, activeTabContent } = useTabs(children);

    const tabHeaderComponents = tabHeaders.map(({ title, isActive, onSelectTab, dataTestId }) => {
        return (
            <NetworkEntityTabHeader
                key={title}
                title={title}
                dataTestId={dataTestId}
                isActive={isActive}
                onSelectTab={onSelectTab}
            />
        );
    });

    return (
        <DetailsOverlay
            headerText={entityName}
            subHeaderText={networkEntityLabels[entityType]}
            tabHeaderComponents={tabHeaderComponents}
            dataTestId="network-entity-tabbed-overlay"
        >
            {activeTabContent}
        </DetailsOverlay>
    );
}

NetworkEntityTabbedOverlay.propTypes = {
    entityName: PropTypes.string.isRequired,
    entityType: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
};

export default NetworkEntityTabbedOverlay;
