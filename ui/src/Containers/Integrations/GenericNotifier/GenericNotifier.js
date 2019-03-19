import React from 'react';
import { Field } from 'redux-form';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';

const removeFieldHandler = (fields, index) => () => {
    fields.remove(index);
};

const addFieldHandler = fields => () => {
    fields.push({});
};

const renderKeyValues = ({ fields }) => (
    <div className="w-full">
        <div className="w-full text-right">
            <button className="text-base-500" onClick={addFieldHandler(fields)} type="button">
                <Icon.PlusSquare size="40" />
            </button>
        </div>
        {fields.map((pair, index) => (
            <div key={pair} className="w-full flex">
                <Field
                    key={`${pair}.key`}
                    name={`${pair}.key`}
                    component="input"
                    type="text"
                    className="border-2 rounded p-2 my-1 mr-1 border-base-300 w-1/3 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                    placeholder="Key"
                />
                <Field
                    key={`${pair}.value`}
                    name={`${pair}.value`}
                    component="input"
                    type="text"
                    className="border-2 rounded p-2 my-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                    placeholder="Value"
                />
                <button
                    className="ml-2 p-2 my-1 rounded-r-sm text-base-100 uppercase text-center text-alert-700 hover:text-alert-800 bg-alert-200 hover:bg-alert-300 border-2 border-alert-300 items-center rounded"
                    onClick={removeFieldHandler(fields, index)}
                    type="button"
                >
                    <Icon.X size="20" />
                </button>
            </div>
        ))}
    </div>
);

renderKeyValues.propTypes = {
    fields: PropTypes.shape({}).isRequired
};

export default renderKeyValues;
