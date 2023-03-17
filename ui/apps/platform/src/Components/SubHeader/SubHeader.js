import React from 'react';
import PropTypes from 'prop-types';

const SubHeader = ({ text, capitalize }) => {
    return <div className={`mt-1 italic ${capitalize ? 'capitalize' : ''}`}>{text}</div>;
};

SubHeader.propTypes = {
    text: PropTypes.string.isRequired,
    capitalize: PropTypes.bool,
};

SubHeader.defaultProps = {
    capitalize: true,
};

export default SubHeader;
