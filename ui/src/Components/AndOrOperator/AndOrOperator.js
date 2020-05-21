import React from 'react';
import PropTypes from 'prop-types';

import BOOLEAN_LOGIC_VALUES from 'constants/booleanLogicValues';

function AndOrOperator({ onToggle, value, disabled, isCircular }) {
    return (
        <div
            className={`flex justify-center ${isCircular && !disabled ? 'py-3' : 'py-2'}`}
            data-testid="and-or-operator"
        >
            <button
                type="button"
                onClick={onToggle}
                disabled={disabled}
                className={`uppercase ${isCircular && !disabled ? 'font-900 text-base-400' : ''}`}
            >
                —
                <span
                    className={`${
                        isCircular && !disabled
                            ? 'border-2 text-base-500 bg-base-300 border-base-400 px-2 rounded-full'
                            : 'p-2'
                    } font-700`}
                >
                    {value}
                </span>
                —
            </button>
        </div>
    );
}

AndOrOperator.propTypes = {
    value: PropTypes.oneOf([BOOLEAN_LOGIC_VALUES.AND, BOOLEAN_LOGIC_VALUES.OR]),
    onToggle: PropTypes.func,
    disabled: PropTypes.bool,
    isCircular: PropTypes.bool,
};

AndOrOperator.defaultProps = {
    value: BOOLEAN_LOGIC_VALUES.OR,
    onToggle: null,
    disabled: false,
    isCircular: false,
};

export default AndOrOperator;
