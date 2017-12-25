import React from 'react';
import PropTypes from 'prop-types';

const TabContent = ({ children, active }) => (
    <div className={active ? 'flex flex-col h-full transition' : 'hidden'}>
        {children}
    </div>
);

TabContent.defaultProps = {
    children: []
};

TabContent.propTypes = {
    children: PropTypes.arrayOf(PropTypes.element),
    active: PropTypes.bool.isRequired
};

export default TabContent;
