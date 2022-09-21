import React from 'react';
import PropTypes from 'prop-types';

/*
 * Follows text in a `label` element and assumes a separating space.
 *
 * The prop value is a comparison, for example `empty={value.length === 0}`
 * If the value is empty: warning color and label weight.
 * If the value is not empty: label color and (assumed) ordinary weight.
 */
const FormFieldRequired = ({ empty }) => (
    <i className={empty ? 'text-warning-700' : 'font-600'} data-testid="required">
        (required)
    </i>
);

FormFieldRequired.propTypes = {
    empty: PropTypes.bool.isRequired,
};

export default FormFieldRequired;
