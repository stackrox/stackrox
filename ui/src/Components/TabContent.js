import React from 'react';
import PropTypes from 'prop-types';

const TabContent = ({ children }) => (
    <div className="flex flex-col h-full transition overflow-auto bg-base-100">{children}</div>
);

TabContent.defaultProps = {
    children: []
};

TabContent.propTypes = {
    children: PropTypes.node
};

export default TabContent;
