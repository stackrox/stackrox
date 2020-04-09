import React from 'react';
import PropTypes from 'prop-types';

import BOOLEAN_LOGIC_VALUES from 'constants/booleanLogicValues';

function AndOrOperator({ onToggle, value }) {
    return (
        <div className="flex justify-center py-2">
            <button type="button" onClick={onToggle} disabled={!onToggle} className="uppercase">
                — {value} —
            </button>
        </div>
    );
}

AndOrOperator.propTypes = {
    value: PropTypes.oneOf([BOOLEAN_LOGIC_VALUES.AND, BOOLEAN_LOGIC_VALUES.OR]),
    onToggle: PropTypes.func
};

AndOrOperator.defaultProps = {
    value: BOOLEAN_LOGIC_VALUES.OR,
    onToggle: null
};

export default AndOrOperator;
