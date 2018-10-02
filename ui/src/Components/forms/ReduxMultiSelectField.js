import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import Select from 'react-select';

const ReduxMultiSelect = props => (
    <Select
        key={props.input.name}
        multi
        onChange={props.input.onChange}
        options={props.options}
        placeholder="Select options"
        removeSelected
        value={props.input.value}
        className="text-base-600 font-400 w-full"
    />
);

ReduxMultiSelect.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
        name: PropTypes.string,
        onChange: PropTypes.func
    }).isRequired,
    options: PropTypes.arrayOf(PropTypes.shape({})).isRequired
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
