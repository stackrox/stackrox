import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import { Creatable } from 'Components/ReactSelect';

const ReduxSelectCreatable = ({
    input: { name, value, onChange },
    options,
    placeholder,
    styles,
    disabled,
}) => (
    <Creatable
        key={name}
        hideSelectedOptions
        onChange={onChange}
        options={options}
        placeholder={placeholder}
        value={value}
        styles={styles}
        isDisabled={disabled}
    />
);

ReduxSelectCreatable.propTypes = {
    input: PropTypes.shape({
        name: PropTypes.string,
        value: PropTypes.string,
        onChange: PropTypes.func,
    }).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    placeholder: PropTypes.string,
    styles: PropTypes.shape({}),
    disabled: PropTypes.bool,
};

ReduxSelectCreatable.defaultProps = {
    placeholder: 'Select options',
    styles: {},
    disabled: false,
};

const ReduxSelectCreatableField = ({ name, options, styles, disabled, placeholder }) => (
    <Field
        key={name}
        name={name}
        options={options}
        component={ReduxSelectCreatable}
        className="border bg-base-100 border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
        styles={styles}
        disabled={disabled}
        placeholder={placeholder}
    />
);

ReduxSelectCreatableField.propTypes = {
    name: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    styles: PropTypes.shape({}),
    disabled: PropTypes.bool,
    placeholder: PropTypes.string,
};

ReduxSelectCreatableField.defaultProps = {
    styles: {},
    disabled: false,
    placeholder: 'Select options',
};

export default ReduxSelectCreatableField;
