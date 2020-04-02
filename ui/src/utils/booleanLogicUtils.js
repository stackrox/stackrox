import BOOLEAN_LOGIC_VALUES from 'constants/booleanLogicValues';

export default function toggleAndOrValue(value) {
    return value === BOOLEAN_LOGIC_VALUES.AND ? BOOLEAN_LOGIC_VALUES.OR : BOOLEAN_LOGIC_VALUES.AND;
}
