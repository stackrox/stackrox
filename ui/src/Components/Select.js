import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

class Select extends Component {
    static propTypes = {
        options: PropTypes.arrayOf(PropTypes.object).isRequired,
        onChange: PropTypes.func.isRequired,
        placeholder: PropTypes.string,
        className: PropTypes.string,
        value: PropTypes.oneOfType([PropTypes.string, PropTypes.number])
    };

    static defaultProps = {
        placeholder: '',
        className:
            'block w-full border bg-base-200 border-base-400 text-base-600 p-3 pr-8 rounded-sm z-1 focus:border-base-500',
        value: ''
    };

    onClick = event => {
        const selectedOption = this.props.options.find(
            option => option.label === event.target.value
        );
        this.props.onChange(selectedOption);
    };

    render() {
        const { className, options, placeholder, value } = this.props;
        return (
            <div className="relative">
                <select
                    className={`${className} cursor-pointer`}
                    onChange={this.onClick}
                    value={value}
                >
                    <option value="" disabled>
                        {placeholder}
                    </option>
                    {options.map(option => (
                        <option key={option.label} value={option.jsonpath}>
                            {option.label}
                        </option>
                    ))}
                </select>
                <div className="absolute pin-y pin-r flex items-center px-2 cursor-pointer z-0 pointer-events-none">
                    <Icon.ChevronDown className="h-4 w-4" />
                </div>
            </div>
        );
    }
}

export default Select;
