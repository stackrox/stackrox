import PropTypes from 'prop-types';
import React from 'react';
import ReduxToggleField from 'Components/forms/ReduxToggleField';

const InlineToggleField = ({ label, name, borderClass }) => (
    <div className={`flex py-2 ${borderClass} border-base-300`} htmlFor={name} key={name}>
        <div className="flex w-full items-center">{label}</div>
        <ReduxToggleField name={name} className="p-0 m-0" />
    </div>
);

InlineToggleField.propTypes = {
    label: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    borderClass: PropTypes.string
};

InlineToggleField.defaultProps = {
    borderClass: ''
};

export default InlineToggleField;
