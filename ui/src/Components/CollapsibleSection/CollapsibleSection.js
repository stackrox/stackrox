import React, { useState } from 'react';
import PropTypes from 'prop-types';
import posed from 'react-pose';

import Button from 'Components/Button';
import { ChevronDown, ChevronRight } from 'react-feather';

const iconClass = 'bg-base-100 border-2 border-base-400 rounded-full h-5 w-5';

const Content = posed.div({
    closed: {
        height: '1px',
        transition: {
            duration: 0,
        },
        flip: true,
    },
    open: {
        height: 'auto',
        transition: {
            duration: 0,
        },
        flip: true,
    },
});

const CollapsibleSection = ({
    title,
    children,
    headerComponents,
    headerClassName,
    titleClassName,
    dataTestId,
}) => {
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
        <div className="border-b border-base-300" data-testid={dataTestId}>
            <header className={`flex flex-1 w-full ${headerClassName}`}>
                <div className="flex flex-1">
                    <div
                        className={`flex py-1 text-base-600 rounded-r-sm font-700 items-center ${titleClassName}`}
                    >
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
    headerClassName: PropTypes.string,
    headerComponents: PropTypes.element,
    titleClassName: PropTypes.string,
    dataTestId: PropTypes.string,
};

CollapsibleSection.defaultProps = {
    headerClassName: 'py-4',
    headerComponents: null,
    titleClassName: 'p-4 text-xl',
    dataTestId: null,
};

export default CollapsibleSection;
