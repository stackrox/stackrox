import React from 'react';
import PropTypes from 'prop-types';
import { Field } from 'formik';
import { Radio } from '@patternfly/react-core';

import { POLICY_DUPE_ACTIONS } from './PolicyImport.utils';

const KeepBothSection = ({ changeRadio }) => {
    return (
        <Field name="resolution">
            {({ field }) => (
                <Radio
                    name={field.name}
                    id="keep-both-radio"
                    value="keepBoth"
                    checked={field.value === POLICY_DUPE_ACTIONS.KEEP_BOTH}
                    onChange={changeRadio(
                        field.onChange,
                        field.name,
                        POLICY_DUPE_ACTIONS.KEEP_BOTH
                    )}
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
