import React from 'react';
import PropTypes from 'prop-types';

const TabContent = ({ children, extraClasses }) => (
    <div className={`${extraClasses} min-h-full overflow-auto`}>{children}</div>
);

TabContent.defaultProps = {
    children: [],
    extraClasses: ''
};

TabContent.propTypes = {
    children: PropTypes.node,
    extraClasses: PropTypes.string
};

export default TabContent;
