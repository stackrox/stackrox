import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import { Creatable } from 'Components/ReactSelect';

const ReduxMultiSelectCreatable = ({
    input: { name, value, onChange },
    options,
    placeholder,
    disabled,
}) => (
    <Creatable
        key={name}
        isMulti
        hideSelectedOptions
        onChange={onChange}
        options={options}
        placeholder={placeholder}
        value={value}
        isDisabled={disabled}
    />
);

ReduxMultiSelectCreatable.propTypes = {
    input: PropTypes.shape({
        name: PropTypes.string,
        value: PropTypes.array,
        onChange: PropTypes.func,
    }).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    placeholder: PropTypes.string,
    disabled: PropTypes.bool,
};

ReduxMultiSelectCreatable.defaultProps = {
    placeholder: 'Select options',
    disabled: false,
};

const ReduxMultiSelectCreatableField = ({ name, options, disabled }) => (
    <Field
        key={name}
        name={name}
        options={options}
        component={ReduxMultiSelectCreatable}
        disabled={disabled}
        className="border bg-base-100 border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
    />
);

ReduxMultiSelectCreatableField.propTypes = {
    name: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    disabled: PropTypes.bool,
};

ReduxMultiSelectCreatableField.defaultProps = {
    disabled: false,
};

export default ReduxMultiSelectCreatableField;
