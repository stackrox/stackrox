import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

const ReduxToggleField = ({ name, disabled, reverse, className }) => {
    const classNames = reverse ? 'form-switch-reverse' : 'form-switch';

    // eslint-disable-next-line jsx-a11y/label-has-associated-control
    const label = <label className="form-switch-label" key={`{name}-label`} htmlFor={name} />;

    return (
        <div className={className}>
            <div className={`${classNames} mr-0 inline-block align-middle h-6`}>
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
        </div>
    );
};

ReduxToggleField.propTypes = {
    name: PropTypes.string.isRequired,
    disabled: PropTypes.bool,
    reverse: PropTypes.bool,
    className: PropTypes.string
};

ReduxToggleField.defaultProps = {
    disabled: false,
    reverse: false,
    className: 'mb-2'
};

export default ReduxToggleField;
