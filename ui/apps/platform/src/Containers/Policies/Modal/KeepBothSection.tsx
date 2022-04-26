import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'formik';
import { Radio } from '@patternfly/react-core';

const KeepBothSection = ({ changeRadio }) => {
    return (
        <Field name="resolution">
            {({ field }) => (
                <Radio
                    name={field.name}
                    id="keep-both-radio"
                    value="keepBoth"
                    checked={field.value === 'keepBoth'}
                    onChange={changeRadio(field.onChange, field.name, 'keepBoth')}
                    label="Keep both policies (imported policy will be assigned a new ID)"
                />
            )}
        </Field>
    );
};

KeepBothSection.propTypes = {
    changeRadio: PropTypes.func.isRequired,
};

export default KeepBothSection;
