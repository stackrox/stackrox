import React from 'react';
import PropTypes from 'prop-types';

const TabContent = ({ children, active }) => (
    <div className={active ? 'flex flex-col h-full transition' : 'hidden'}>
        {children}
    </div>
);

TabContent.defaultProps = {
    children: [],
    active: false
};

TabContent.propTypes = {
    children: PropTypes.node,
    active: PropTypes.bool
};

export default TabContent;
