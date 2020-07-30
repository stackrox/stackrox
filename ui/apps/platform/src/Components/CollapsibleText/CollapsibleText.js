import React, { useState } from 'react';
import PropTypes from 'prop-types';

import Button from 'Components/Button';
import CollapsibleAnimatedDiv from 'Components/animations/CollapsibleAnimatedDiv';

const CollapsibleText = ({ initiallyExpanded, children }) => {
    const [open, setOpen] = useState(initiallyExpanded);

    function toggleOpen() {
        setOpen(!open);
    }

    const text = open ? 'Show less' : 'Show more...';

    return (
        <div>
            <CollapsibleAnimatedDiv isOpen={open}>{children}</CollapsibleAnimatedDiv>
            <Button text={text} onClick={toggleOpen} className="hover:text-primary-700 underline" />
        </div>
    );
};

CollapsibleText.propTypes = {
    initiallyExpanded: PropTypes.bool,
    children: PropTypes.node.isRequired,
};

CollapsibleText.defaultProps = {
    initiallyExpanded: false,
};

export default CollapsibleText;
