import React from 'react';
import { FormGroup, FormGroupProps, Radio } from '@patternfly/react-core';

import { useFormik } from 'formik';
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { DeferralValues, ScopeContext } from './utils';

export const ALL = '.*';

export type ExceptionScopeFieldProps = {
    fieldId: FormGroupProps['fieldId'];
    label: FormGroupProps['label'];
    scopeContext: ScopeContext;
    formik: ReturnType<typeof useFormik<DeferralValues>>;
};

function ExceptionScopeField({ fieldId, label, scopeContext, formik }: ExceptionScopeFieldProps) {
    const { values } = formik;

    return (
        <FormGroup fieldId={fieldId} label={label} isRequired>
            {scopeContext === 'GLOBAL' && (
                <Radio
                    id="scope-global"
                    name="scope-global"
                    isDisabled
                    isChecked={
                        values.scope.imageScope.registry === ALL &&
                        values.scope.imageScope.remote === ALL &&
                        values.scope.imageScope.tag === ALL
                    }
                    onChange={() => {}}
                    label="Selected CVEs across all images and deployments"
                />
            )}
            {scopeContext !== 'GLOBAL' && (
                <>
                    <Radio
                        id="scope-single-image"
                        name="scope-single-image"
                        isChecked={
                            values.scope.imageScope.registry === ALL &&
                            values.scope.imageScope.remote === scopeContext.image.name &&
                            values.scope.imageScope.tag === ALL
                        }
                        onChange={() => {}}
                        label={`All tags within ${scopeContext.image.name}}`}
                    />
                    <Radio
                        id="scope-single-image-single-tag"
                        name="scope-single-image-single-tag"
                        isChecked={
                            values.scope.imageScope.registry === ALL &&
                            values.scope.imageScope.remote === scopeContext.image.name &&
                            values.scope.imageScope.tag === scopeContext.image.tag
                        }
                        label={`Only ${scopeContext.image.name}:${scopeContext.image.tag}`}
                    />
                </>
            )}
        </FormGroup>
    );
}

export default ExceptionScopeField;
