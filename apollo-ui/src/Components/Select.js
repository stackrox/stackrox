import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const Select = ({
    className,
    options,
    placeholder,
    onChange,
    value
}) => {
    this.handleClick = (event) => {
        onChange(event.target.value);
    };
    return (
        <div className="relative">
            <select
                className={className}
                onChange={this.handleClick}
                value={value}
            >
                <option value="" disabled>{placeholder}</option>
                {options.map(option => <option key={option.label} value={option.value}>{option.label}</option>)}
            </select>
            <div className="absolute pin-y pin-r flex items-center px-2 cursor-pointer z-0">
                <Icon.ChevronDown className="h-4 w-4" />
            </div>
        </div>
    );
};

Select.defaultProps = {
    placeholder: '',
    className: 'block w-full border bg-white border-base-300 text-base-600 p-3 pr-8 rounded cursor-pointer z-1 focus:border-base-300'
};

Select.propTypes = {
    options: PropTypes.arrayOf(PropTypes.object).isRequired,
    onChange: PropTypes.func.isRequired,
    placeholder: PropTypes.string,
    className: PropTypes.string
};

export default Select;
