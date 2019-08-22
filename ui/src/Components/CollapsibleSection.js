import React, { useState } from 'react';
import PropTypes from 'prop-types';
import posed from 'react-pose';

import Button from 'Components/Button';
import { ChevronDown, ChevronRight } from 'react-feather';

const iconClass = 'bg-base-100 border-2 border-base-400 rounded-full h-4 w-4';

const Content = posed.div({
    closed: { height: 0 },
    open: { height: 'inherit' }
});

const CollapsibleSection = ({ title, children, headerComponents }) => {
    const [open, setOpen] = useState(true);

    function toggleOpen() {
        setOpen(!open);
    }

    const Icon = open ? (
        <ChevronDown className={iconClass} />
    ) : (
        <ChevronRight className={iconClass} />
    );

    return (
        <div className="border-b border-base-300">
            <header className="flex flex-1 w-full py-4">
                <div className="flex flex-1">
                    <div className="flex px-4 py-1 text-base-600 rounded-r-sm font-700 text-xl items-center">
                        <Button icon={Icon} onClick={toggleOpen} />
                        <span className="ml-2">{title}</span>
                    </div>
                </div>
                {headerComponents}
            </header>
            <Content className={open ? '' : 'overflow-hidden'} pose={open ? 'open' : 'closed'}>
                {children}
            </Content>
        </div>
    );
};

CollapsibleSection.propTypes = {
    title: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
    headerComponents: PropTypes.element
};

CollapsibleSection.defaultProps = {
    headerComponents: null
};

export default CollapsibleSection;
