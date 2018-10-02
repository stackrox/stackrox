import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import Select from 'Components/ReactSelect';

const ReduxMultiSelect = ({ input: { name, value, onChange }, options, placeholder }) => (
    <Select
        key={name}
        isMulti
        onChange={onChange}
        options={options}
        placeholder={placeholder}
        hideSelectedOptions
        value={value}
    />
);

ReduxMultiSelect.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
        name: PropTypes.string,
        onChange: PropTypes.func
    }).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    placeholder: PropTypes.string
};

ReduxMultiSelect.defaultProps = {
    placeholder: 'Select options'
};

const ReduxMultiSelectField = ({ name, options }) => (
    <Field
        key={name}
        name={name}
        options={options}
        component={ReduxMultiSelect}
        className="border bg-base-100 border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
    />
);

ReduxMultiSelectField.propTypes = {
    name: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

export default ReduxMultiSelectField;
