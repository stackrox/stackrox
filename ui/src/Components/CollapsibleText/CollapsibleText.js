import React, { useState } from 'react';
import PropTypes from 'prop-types';
import posed from 'react-pose';
import Button from 'Components/Button';

const Content = posed.div({
    closed: { height: 0 },
    open: { height: 'inherit' }
});

const CollapsibleText = ({ initiallyExpanded, expandText, collapseText, children }) => {
    const [open, setOpen] = useState(initiallyExpanded);

    function toggleOpen() {
        setOpen(!open);
    }

    return (
        <div>
            <Content className={open ? '' : 'overflow-hidden'} pose={open ? 'open' : 'closed'}>
                {children}
                <Button
                    text={collapseText}
                    onClick={toggleOpen}
                    className="hover:text-primary-700 underline"
                />
            </Content>
            <Content className={open ? 'overflow-hidden' : ''} pose={open ? 'closed' : 'open'}>
                <Button
                    text={expandText}
                    onClick={toggleOpen}
                    className="hover:text-primary-700 underline"
                />
            </Content>
        </div>
    );
};

CollapsibleText.propTypes = {
    initiallyExpanded: PropTypes.bool,
    expandText: PropTypes.string,
    collapseText: PropTypes.string,
    children: PropTypes.node.isRequired
};

CollapsibleText.defaultProps = {
    initiallyExpanded: false,
    expandText: 'Show more ...',
    collapseText: 'Show less'
};

export default CollapsibleText;
