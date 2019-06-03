import React, { useState } from 'react';
import PropTypes from 'prop-types';
import posed from 'react-pose';

import Button from 'Components/Button';

const Content = posed.div({
    closed: { height: 0 },
    open: { height: 'inherit' }
});

const CollapsibleSection = ({ title, children }) => {
    const [open, setOpen] = useState(true);

    function toggleOpen() {
        setOpen(!open);
    }

    return (
        <div className="border-b border-base-300">
            <header className="flex flex-1 w-full py-4">
                <div className="flex flex-1">
                    <div className="flex px-4 py-1 bg-primary-400 text-base-100 rounded-r-sm font-700 items-center">
                        {title}
                    </div>
                </div>
                <Button
                    className="border border-base-300 px-4 py-1 text-base-600 text-sm justify-end mr-4"
                    text="Collapse"
                    onClick={toggleOpen}
                />
            </header>
            <Content className="overflow-hidden" pose={open ? 'open' : 'closed'}>
                {children}
            </Content>
        </div>
    );
};

CollapsibleSection.propTypes = {
    title: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired
};

export default CollapsibleSection;
