/* eslint-disable jsx-a11y/label-has-associated-control */
/**
 * disabled the rule above, because we are using an extra <label> element
 *   as the visual "slider" in the toggle
 */

import React from 'react';
import PropTypes from 'prop-types';
import { set } from 'lodash';

function ToggleSwitch({
    id,
    toggleHandler,
    label,
    enabled,
    extraClassNames,
    flipped,
    small,
    disabled
}) {
    const flippedToggleHandler = e => {
        set(e, 'target.checked', !e.target.checked);
        toggleHandler(e);
    };
    return (
        <div className={`toggle-switch-wrapper ${extraClassNames}`}>
            <label className="text-xs text-grey-dark" htmlFor={id}>
                {label}
            </label>
            <div
                className={`toggle-switch inline-block align-middle ml-2 ${
                    small ? 'toggle-switch-small' : ''
                }`}
            >
                <input
                    type="checkbox"
                    checked={flipped ? !enabled : !!enabled}
                    onChange={flipped ? flippedToggleHandler : toggleHandler}
                    name={id}
                    id={id}
                    disabled={disabled}
                    className="toggle-switch-checkbox"
                />
                <label className="toggle-switch-label" htmlFor={id} />
            </div>
        </div>
    );
}

ToggleSwitch.propTypes = {
    id: PropTypes.string.isRequired,
    toggleHandler: PropTypes.func.isRequired,
    label: PropTypes.string,
    enabled: PropTypes.bool,
    extraClassNames: PropTypes.string,
    flipped: PropTypes.bool,
    small: PropTypes.bool,
    disabled: PropTypes.bool
};

ToggleSwitch.defaultProps = {
    label: '',
    enabled: false,
    extraClassNames: '',
    flipped: false,
    small: false,
    disabled: false
};

export default ToggleSwitch;

/* eslint-enable jsx-a11y/label-has-associated-control */
