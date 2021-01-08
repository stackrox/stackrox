import React from 'react';
import PropTypes from 'prop-types';

import useTabs from 'hooks/useTabs';
import { networkEntityLabels } from 'messages/network';

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
        <div
            className="flex flex-1 flex-col text-sm network-overlay-bg-shadow max-h-minus-buttons w-168"
            data-testid="network-details-panel"
        >
            <div
                className="bg-primary-800 flex items-center m-2 p-3 rounded-lg shadow text-primary-100"
                data-testid="network-details-panel-header"
            >
                <div className="flex flex-1 flex-col">
                    <div className="text-base">{entityName}</div>
                    <div className="italic text-primary-300 text-xs capitalize">
                        {networkEntityLabels[entityType]}
                    </div>
                </div>
                <ul className="flex ml-8 items-center text-sm uppercase font-700">
                    {tabHeaderComponents}
                </ul>
            </div>
            <div className="flex flex-1 m-2 overflow-auto rounded">{activeTabContent}</div>
        </div>
    );
}

NetworkEntityTabbedOverlay.propTypes = {
    entityName: PropTypes.string.isRequired,
    entityType: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
};

export default NetworkEntityTabbedOverlay;
