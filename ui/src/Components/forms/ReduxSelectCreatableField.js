import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import { Creatable } from 'Components/ReactSelect';

const ReduxSelectCreatable = ({
    input: { name, value, onChange },
    options,
    placeholder,
    styles
}) => (
    <Creatable
        key={name}
        hideSelectedOptions
        onChange={onChange}
        options={options}
        placeholder={placeholder}
        value={value}
        styles={styles}
    />
);

ReduxSelectCreatable.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.string,
        onChange: PropTypes.func
    }).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    placeholder: PropTypes.string,
    styles: PropTypes.shape({})
};

ReduxSelectCreatable.defaultProps = {
    placeholder: 'Select options',
    styles: {}
};

const ReduxSelectCreatableField = ({ name, options, styles }) => (
    <Field
        key={name}
        name={name}
        options={options}
        component={ReduxSelectCreatable}
        className="border bg-base-100 border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
        styles={styles}
    />
);

ReduxSelectCreatableField.propTypes = {
    name: PropTypes.string.isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    styles: PropTypes.shape({})
};

ReduxSelectCreatableField.defaultProps = {
    styles: {}
};

export default ReduxSelectCreatableField;
