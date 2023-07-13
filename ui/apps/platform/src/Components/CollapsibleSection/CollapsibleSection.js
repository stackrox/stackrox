import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { ChevronDown, ChevronRight } from 'react-feather';

import Button from 'Components/Button';
import CollapsibleAnimatedDiv from 'Components/animations/CollapsibleAnimatedDiv';

const iconClass = 'bg-base-100 border-2 border-base-400 rounded-full h-5 w-5';

const CollapsibleSection = ({
    id,
    title,
    children,
    headerComponents,
    headerClassName,
    titleClassName,
    dataTestId,
    defaultOpen,
}) => {
    const [isOpen, setIsOpen] = useState(defaultOpen);

    function toggleOpen() {
        setIsOpen(!isOpen);
    }

    const Icon = isOpen ? (
        <ChevronDown className={iconClass} />
    ) : (
        <ChevronRight className={iconClass} />
    );

    return (
        <div id={id} className="border-b border-base-300" data-testid={dataTestId}>
            <header className={`flex flex-1 w-full ${headerClassName}`}>
                <div className="flex flex-1">
                    <div
                        className={`flex py-1 text-base-600 rounded-r-sm font-700 items-center ${titleClassName}`}
                    >
                        <Button
                            icon={Icon}
                            onClick={toggleOpen}
                            aria-label={isOpen ? 'Collapse' : 'Expand'}
                            aria-expanded={isOpen}
                        />
                        <span className="ml-2">{title}</span>
                    </div>
                </div>
                {headerComponents}
            </header>
            <CollapsibleAnimatedDiv defaultOpen isOpen={isOpen} dataTestId="collapsible-content">
                {children}
            </CollapsibleAnimatedDiv>
        </div>
    );
};

CollapsibleSection.propTypes = {
    id: PropTypes.string,
    title: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired,
    headerClassName: PropTypes.string,
    headerComponents: PropTypes.element,
    titleClassName: PropTypes.string,
    dataTestId: PropTypes.string,
    defaultOpen: PropTypes.bool,
};

CollapsibleSection.defaultProps = {
    id: null,
    headerClassName: 'py-4',
    headerComponents: null,
    titleClassName: 'p-4 text-lg',
    dataTestId: null,
    defaultOpen: true,
};

export default CollapsibleSection;
