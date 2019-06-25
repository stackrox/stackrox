import React from 'react';
import PropTypes from 'prop-types';

const LabelChip = ({ text, type }) => {
    let colorClassName = '';
    if (type === 'alert') colorClassName = 'bg-alert-200 border-alert-400 text-alert-800';
    return <span className={`border px-2 rounded ${colorClassName}`}>{text}</span>;
};

LabelChip.propTypes = {
    text: PropTypes.string.isRequired,
    type: PropTypes.string.isRequired
};

export default LabelChip;
