import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxCheckboxField = ({ name, disabled }) => (
    <Field key={name} name={name} component="input" type="checkbox" disabled={disabled} />
);

ReduxCheckboxField.propTypes = {
    name: PropTypes.string.isRequired,
    disabled: PropTypes.bool
};

ReduxCheckboxField.defaultProps = {
    disabled: false
};

export default ReduxCheckboxField;
