import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

import RadioButtonGroup from 'Components/RadioButtonGroup';

function ReduxRadioButtonGroup({ input, buttons, groupClassName, useBoolean, disabled }) {
    const { value, onChange } = input;
    return (
        <RadioButtonGroup
            buttons={buttons}
            onClick={onChange}
            selected={value}
            groupClassName={groupClassName}
            useBoolean={useBoolean}
            disabled={disabled}
        />
    );
}

function ReduxRadioButtonGroupField({ name, buttons, groupClassName, useBoolean, disabled }) {
    return (
        <Field
            key={name}
            name={name}
            id={name}
            component={ReduxRadioButtonGroup}
            buttons={buttons}
            groupClassName={groupClassName}
            useBoolean={useBoolean}
            disabled={disabled}
        />
    );
}

ReduxRadioButtonGroupField.propTypes = {
    name: PropTypes.string.isRequired,
    buttons: PropTypes.arrayOf(
        PropTypes.shape({
            text: PropTypes.string.isRequired,
            value: PropTypes.bool,
        })
    ).isRequired,
    groupClassName: PropTypes.string,
    useBoolean: PropTypes.bool,
    disabled: PropTypes.bool,
};

ReduxRadioButtonGroupField.defaultProps = {
    groupClassName: '',
    useBoolean: false,
    disabled: false,
};

export default ReduxRadioButtonGroupField;
