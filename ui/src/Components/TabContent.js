import React from 'react';
import PropTypes from 'prop-types';

const TabContent = ({ children }) => <div className="h-full overflow-auto">{children}</div>;

TabContent.defaultProps = {
    children: []
};

TabContent.propTypes = {
    children: PropTypes.node
};

export default TabContent;
