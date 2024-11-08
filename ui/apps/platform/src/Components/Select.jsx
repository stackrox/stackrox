import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

class Select extends Component {
    static propTypes = {
        options: PropTypes.arrayOf(
            PropTypes.shape({
                label: PropTypes.string,
                value: PropTypes.string,
            })
        ).isRequired,
        onChange: PropTypes.func.isRequired,
        placeholder: PropTypes.string,
        className: PropTypes.string,
        wrapperClass: PropTypes.string,
        triggerClass: PropTypes.string,
        value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
        disabled: PropTypes.bool,
    };

    static defaultProps = {
        placeholder: '',
        className:
            'block w-full border bg-base-200 border-base-400 text-base-600 p-3 pr-8 rounded-sm z-1 focus:border-base-500',
        wrapperClass: '',
        triggerClass: '',
        value: '',
        disabled: false,
    };

    onClick = (event) => {
        const selectedOption = this.props.options.find(
            (option) => option.value === event.target.value
        );
        if (!selectedOption) {
            throw new Error('Selected ID does not match any known option in Select control.');
        }

        this.props.onChange(selectedOption);
    };

    render() {
        const { className, wrapperClass, triggerClass, options, placeholder, value, disabled } =
            this.props;
        return (
            <div className={`flex relative ${wrapperClass}`}>
                <select
                    className={`${className} pr-8 w-full cursor-pointer`}
                    onChange={this.onClick}
                    value={value}
                    aria-label={placeholder}
                    disabled={disabled}
                >
                    {placeholder && (
                        <option value="" disabled>
                            {placeholder}
                        </option>
                    )}
                    {options.map((option) => (
                        <option key={option.label} value={option.value}>
                            {option.label}
                        </option>
                    ))}
                </select>
                <div
                    className={`${triggerClass} absolute inset-y-0 right-0 flex items-center px-2 cursor-pointer z-10 pointer-events-none`}
                >
                    <Icon.ChevronDown className="h-4 w-4" />
                </div>
            </div>
        );
    }
}

export default Select;
