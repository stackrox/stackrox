import React, { useState } from 'react';
import PropTypes from 'prop-types';
import posed from 'react-pose';

import { ChevronRight, ChevronDown } from 'react-feather';

const icons = {
    opened: (
        <ChevronDown className="bg-base-200 border border-base-400 mr-4 rounded-full" size="14" />
    ),
    closed: (
        <ChevronRight className="bg-base-200 border border-base-400 mr-4 rounded-full" size="14" />
    )
};

const Content = posed.div({
    closed: { height: 0 },
    open: { height: 'inherit' }
});

const CollapsibleRow = ({ header, children }) => {
    const [open, setOpen] = useState(true);

    function toggleOpen() {
        setOpen(!open);
    }

    return (
        <div className="border-b border-base-300">
            <button
                type="button"
                className="flex flex-1 w-full cursor-pointer hover:bg-primary-100"
                onClick={toggleOpen}
            >
                <div className={`flex w-full p-3 ${open ? 'border-b border-base-300' : ''}`}>
                    {icons[open ? 'opened' : 'closed']}
                    {header}
                </div>
            </button>
            <Content className="overflow-hidden" pose={open ? 'open' : 'closed'}>
                {children}
            </Content>
        </div>
    );
};

CollapsibleRow.propTypes = {
    header: PropTypes.node.isRequired,
    children: PropTypes.node.isRequired
};

export default CollapsibleRow;
