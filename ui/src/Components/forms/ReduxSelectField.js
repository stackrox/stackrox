import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import Select from 'react-select';

const ReduxSelect = props => (
    <Select
        key={props.input.name}
        onChange={props.input.onChange}
        options={props.options}
        placeholder="Select one..."
        simpleValue
        value={props.input.value}
        className="text-base-600 font-400 w-full"
    />
);

ReduxSelect.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]),
        name: PropTypes.string,
        onChange: PropTypes.func
    }).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

const ReduxSelectField = ({ name, options }) => (
    <Field
        key={name}
        name={name}
        options={options}
        component={ReduxSelect}
        className="border bg-white border-base-300 text-base-600 p-3 pr-8 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-full font-400"
    />
);

ReduxSelectField.propTypes = {
    name: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

export default ReduxSelectField;
