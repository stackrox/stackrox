import React from 'react';
import PropTypes from 'prop-types';

import useTabs from 'hooks/useTabs';
import { networkEntityLabels } from 'messages/network';

import NetworkEntityTabHeader from './NetworkEntityTabHeader';

function NetworkEntityTabbedOverlay({ entityName, entityType, children }) {
    const { tabHeaders, activeTabContent } = useTabs(children);

    const tabHeaderComponents = tabHeaders.map(({ title, isActive, onSelectTab }) => {
        return (
            <NetworkEntityTabHeader
                key={title}
                title={title}
                isActive={isActive}
                onSelectTab={onSelectTab}
            />
        );
    });

    return (
        <>
            <div className="max-w-120 bg-primary-800 flex items-center m-2 min-w-108 p-3 rounded-lg shadow text-primary-100">
                <div className="flex flex-1 flex-col">
                    <div>{entityName}</div>
                    <div className="italic text-primary-200 text-xs capitalize">
                        {networkEntityLabels[entityType]}
                    </div>
                </div>
                <ul className="flex ml-8 items-center text-sm uppercase font-700">
                    {tabHeaderComponents}
                </ul>
            </div>
            <div className="flex flex-1 m-2 max-w-120 overflow-auto rounded">
                {activeTabContent}
            </div>
        </>
    );
}

NetworkEntityTabbedOverlay.propTypes = {
    entityName: PropTypes.string.isRequired,
    entityType: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
};

export default NetworkEntityTabbedOverlay;
