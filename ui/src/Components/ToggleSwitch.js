/* eslint-disable jsx-a11y/label-has-associated-control */
/**
 * disabled the rule above, because we are using an extra <label> element
 *   as the visual "slider" in the toggle
 */

import React from 'react';
import PropTypes from 'prop-types';

function ToggleSwitch({ id, toggleHandler, label, enabled, extraClassNames }) {
    return (
        <div className={`toggle-switch-wrapper mb-2 ${extraClassNames}`}>
            <label className="text-xs text-grey-dark" htmlFor={id}>
                {label}
            </label>
            <div className="toggle-switch inline-block align-middle ml-2">
                <input
                    type="checkbox"
                    checked={!!enabled}
                    onChange={toggleHandler}
                    name={id}
                    id={id}
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
    label: PropTypes.string.isRequired,
    enabled: PropTypes.bool,
    extraClassNames: PropTypes.string
};

ToggleSwitch.defaultProps = {
    enabled: false,
    extraClassNames: ''
};

export default ToggleSwitch;

/* eslint-enable jsx-a11y/label-has-associated-control */
