/* eslint-disable react-hooks/rules-of-hooks */
import React, { useState } from 'react';

import BOOLEAN_LOGIC_VALUES from 'constants/booleanLogicValues';
import toggleBooleanValue from 'utils/booleanLogicUtils';
import AndOrOperator from './AndOrOperator';

export default {
    title: 'AndOrOperator',
    component: AndOrOperator
};

export const withAnd = () => {
    const [currentOperator, setCurrentOperator] = useState(BOOLEAN_LOGIC_VALUES.AND);

    function onToggle() {
        setCurrentOperator(toggleBooleanValue(currentOperator));
    }

    return <AndOrOperator value={currentOperator} onToggle={onToggle} />;
};

export const withOr = () => {
    const [currentOperator, setCurrentOperator] = useState(BOOLEAN_LOGIC_VALUES.OR);

    function onToggle() {
        setCurrentOperator(toggleBooleanValue(currentOperator));
    }
    return <AndOrOperator value={currentOperator} onToggle={onToggle} />;
};

export const withAndOnly = () => <AndOrOperator value={BOOLEAN_LOGIC_VALUES.AND} />;

export const withOrOnly = () => <AndOrOperator value={BOOLEAN_LOGIC_VALUES.OR} />;
