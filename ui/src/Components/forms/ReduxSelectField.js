import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import Select from 'Components/ReactSelect';

const ReduxSelect = ({ input: { name, value, onChange }, options, placeholder }) => (
    <Select
        key={name}
        onChange={onChange}
        options={options}
        placeholder={placeholder}
        value={value}
    />
);

ReduxSelect.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]),
        name: PropTypes.string,
        onChange: PropTypes.func
    }).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    placeholder: PropTypes.string.isRequired
};

const ReduxSelectField = ({ name, options, placeholder }) => (
    <Field
        key={name}
        name={name}
        options={options}
        component={ReduxSelect}
        placeholder={placeholder}
        className="border bg-base-100 border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
    />
);

ReduxSelectField.propTypes = {
    name: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    placeholder: PropTypes.string
};

ReduxSelectField.defaultProps = {
    placeholder: 'Select one...'
};

export default ReduxSelectField;
