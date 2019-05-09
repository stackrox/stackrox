import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxToggleField = ({ name, disabled, reverse }) => {
    const classNames = reverse ? 'form-switch-reverse' : 'form-switch';

    // eslint-disable-next-line jsx-a11y/label-has-associated-control
    const label = <label className="form-switch-label" key={`{name}-label`} htmlFor={name} />;

    return (
        <div className={`${classNames} mr-0 inline-block align-middle`}>
            <Field
                key={name}
                id={name}
                name={name}
                component="input"
                type="checkbox"
                className="form-switch-checkbox"
                disabled={disabled}
            />
            {label}
        </div>
    );
};

ReduxToggleField.propTypes = {
    name: PropTypes.string.isRequired,
    disabled: PropTypes.bool,
    reverse: PropTypes.bool
};

ReduxToggleField.defaultProps = {
    disabled: false,
    reverse: false
};

export default ReduxToggleField;
