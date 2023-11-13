import React, { useCallback, ReactElement } from 'react';
import { Form, Radio } from '@patternfly/react-core';
import { Field } from 'formik';

import RenamePolicySection from './RenamePolicySection';
import KeepBothSection from './KeepBothSection';

type DuplicatePolicyFormProps = {
    updateResolution: (name: string, value: string) => void;
    showKeepBothPolicies: boolean;
};

function DuplicatePolicyForm({
    updateResolution,
    showKeepBothPolicies,
}: DuplicatePolicyFormProps): ReactElement {
    // this creates a partially applied function to update the radio button value,
    //   and then notified the parent
    const changeRadio = useCallback(
        (handler: any, name: any, value: any) => () => {
            handler(name)(value);
            updateResolution(name, value);
        },
        [updateResolution]
    );

    // this creates a partially applied function to update a text value,
    //   and then notified the parent
    const changeText = useCallback(
        (handler: any, name: any) => (value) => {
            handler(name)(value);
            updateResolution(name, value);
        },
        [updateResolution]
    );

    return (
        <Form data-testid="dupe-policy-form" className="pf-u-mt-md">
            {!showKeepBothPolicies && (
                <RenamePolicySection changeRadio={changeRadio} changeText={changeText} />
            )}
            {showKeepBothPolicies && <KeepBothSection changeRadio={changeRadio} />}
            <Field name="resolution">
                {({ field }) => (
                    <Radio
                        name={field.name}
                        value="overwrite"
                        label="Overwrite existing policy"
                        id="policy-overwrite-radio-1"
                        isChecked={field.value === 'overwrite'}
                        onChange={changeRadio(field.onChange, field.name, 'overwrite')}
                    />
                )}
            </Field>
        </Form>
    );
}

export default DuplicatePolicyForm;
