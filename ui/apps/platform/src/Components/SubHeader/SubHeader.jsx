import React from 'react';
import PropTypes from 'prop-types';

const SubHeader = ({ text }) => {
    return <div>{text}</div>;
};

SubHeader.propTypes = {
    text: PropTypes.string.isRequired,
};

export default SubHeader;
