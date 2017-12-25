import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const Select = ({ options }) => (
    <div className="relative ml-3">
        <select className="block w-full border bg-base-100 border-base-200 text-base-500 p-3 pr-8 rounded">
            {options.map(option => <option key={option}>{option}</option>)}
        </select>
        <div className="absolute pin-y pin-r flex items-center px-2">
            <Icon.ChevronDown className="h-4 w-4" />
        </div>
    </div>
);

Select.propTypes = {
    options: PropTypes.arrayOf(PropTypes.string).isRequired,
};

export default Select;
