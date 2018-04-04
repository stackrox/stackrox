import React from 'react';
import PropTypes from 'prop-types';

const TabContent = ({ children }) => (
    <div className="flex flex-col h-full transition overflow-auto">{children}</div>
);

TabContent.defaultProps = {
    children: []
};

TabContent.propTypes = {
    children: PropTypes.node
};

export default TabContent;
