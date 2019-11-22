import React from 'react';
import PropTypes from 'prop-types';

const HoverHintListItem = ({ label, value }) => (
    <li className="py-1 list-reset text-base-600 text-xs" key="categories">
        <span className="font-700 mr-1">{label}:</span>
        <span className="font-500">{value}</span>
    </li>
);

HoverHintListItem.propTypes = {
    label: PropTypes.node.isRequired,
    value: PropTypes.node.isRequired
};

export default HoverHintListItem;
