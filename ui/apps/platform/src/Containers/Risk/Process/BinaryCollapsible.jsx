import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import CollapsibleCard from 'Components/CollapsibleCard';

const titleClassName =
    'border-b border-base-300 leading-normal cursor-pointer justify-between items-center hover:border-primary-300';
function BinaryCollapsible({ commandLineArgs, children }) {
    function renderHeader(backgroundClass, icon) {
        let displayArgs = commandLineArgs;
        if (commandLineArgs === '') {
            displayArgs = 'No Arguments';
        }
        return (
            <div className={`${titleClassName} ${backgroundClass}`}>
                <div className="flex items-center">
                    <div className="flex pl-2">{icon}</div>
                    <div className="p-2 flex flex-1">
                        <span className="text-base word-break">{displayArgs}</span>
                    </div>
                </div>
            </div>
        );
    }

    function renderWhenOpened() {
        return renderHeader('bg-primary-100', <Icon.ChevronUp className="h-4 w-4" />);
    }

    function renderWhenClosed() {
        return renderHeader('bg-base-100', <Icon.ChevronDown className="h-4 w-4" />);
    }

    return (
        <CollapsibleCard
            title={commandLineArgs}
            open
            renderWhenOpened={renderWhenOpened}
            renderWhenClosed={renderWhenClosed}
            cardClassName=""
        >
            {children}
        </CollapsibleCard>
    );
}

BinaryCollapsible.propTypes = {
    commandLineArgs: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
};

export default BinaryCollapsible;
