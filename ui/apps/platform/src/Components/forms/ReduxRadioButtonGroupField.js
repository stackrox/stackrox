import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';

import RadioButtonGroup from 'Components/RadioButtonGroup';

function ReduxRadioButtonGroup({ input, buttons, groupClassName, useBoolean, disabled, readonly }) {
    const { value, onChange } = input;
    const onChangeEnabled = readonly ? () => {} : onChange;

    return (
        <RadioButtonGroup
            buttons={buttons}
            // eslint-disable-next-line react/jsx-no-bind
            onClick={onChangeEnabled}
            selected={value}
            groupClassName={groupClassName}
            useBoolean={useBoolean}
            disabled={disabled}
        />
    );
}

function ReduxRadioButtonGroupField({
    name,
    buttons,
    groupClassName,
    useBoolean,
    disabled,
    readonly,
}) {
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
            readonly={readonly}
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
    readonly: PropTypes.bool,
};

ReduxRadioButtonGroupField.defaultProps = {
    groupClassName: '',
    useBoolean: false,
    disabled: false,
    readonly: false,
};

export default ReduxRadioButtonGroupField;
