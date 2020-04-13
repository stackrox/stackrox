import React from 'react';

import Widget from 'Components/Widget';

import removeEmptyFields from 'utils/removeEmptyFields';
import fieldsMap from 'Containers/Policies/Wizard/Details/descriptors';

const PolicyConfigurationFields = ({ fields, ...rest }) => {
    if (!fields) return null;

    const paredFields = removeEmptyFields(fields);
    const fieldKeys = Object.keys(paredFields);

    const fieldList = fieldKeys.map(key => {
        if (!fieldsMap[key]) return '';
        const { label } = fieldsMap[key];
        const value = fieldsMap[key].formatValue(paredFields[key]);
        return (
            <li className="border-b border-base-300 py-2" key={key} data-testid={key}>
                <div className="text-base-600 font-700">{label}:</div>
                <div className="flex pt-1 leading-normal">{value}</div>
            </li>
        );
    });

    return (
        <Widget header="Policy Criteria" {...rest}>
            <div className="flex flex-col w-full">
                <div className="flex w-full h-full text-sm">
                    <ul className="flex-1 border-base-300 overflow-hidden px-2">{fieldList}</ul>
                </div>
            </div>
        </Widget>
    );
};

export default PolicyConfigurationFields;
