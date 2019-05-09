import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'redux-form';
import ColorPicker from 'Components/ColorPicker';

const ReduxColorPicker = ({ input }) => (
    <ColorPicker color={input.value} onChange={input.onChange} />
);

ReduxColorPicker.propTypes = {
    input: PropTypes.shape({
        value: PropTypes.string,
        onChange: PropTypes.func
    }).isRequired
};

const ReduxColorPickerField = ({ name }) => (
    <Field key={name} id={name} name={name} component={ReduxColorPicker} />
);

ReduxColorPickerField.propTypes = {
    name: PropTypes.string.isRequired
};

export default ReduxColorPickerField;
