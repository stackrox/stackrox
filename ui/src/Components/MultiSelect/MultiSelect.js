import React from 'react';
import PropTypes from 'prop-types';

import ReactSelect from 'Components/ReactSelect';

const MultiSelect = ({
    name,
    value,
    onChange,
    options,
    placeholder,
    className,
    'data-testid': dataTestId, // see https://stackoverflow.com/a/51613732
    wrapperClass,
    triggerClass
}) => (
    <ReactSelect
        key={name}
        isMulti
        hideSelectedOptions
        onChange={onChange}
        options={options}
        placeholder={placeholder}
        value={value}
        className={className}
        inputId={dataTestId}
        wrapperClass={wrapperClass}
        triggerClass={triggerClass}
    />
);

MultiSelect.propTypes = {
    name: PropTypes.string.isRequired,
    value: PropTypes.arrayOf(PropTypes.any).isRequired,
    onChange: PropTypes.func.isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    placeholder: PropTypes.string,
    className: PropTypes.string,
    'data-testid': PropTypes.string,
    wrapperClass: PropTypes.string,
    triggerClass: PropTypes.string
};

MultiSelect.defaultProps = {
    placeholder: 'Select options',
    className:
        'block w-full border bg-base-200 border-base-400 text-base-600 p-3 pr-8 rounded-sm z-1 focus:border-base-500',
    'data-testid': '',
    wrapperClass: '',
    triggerClass: ''
};

export default MultiSelect;
