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
    const { values, setFieldValue } = formik;

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
                    label="Selected CVEs across all images and deployments"
                />
            )}
            {scopeContext !== 'GLOBAL' && (
                <>
                    <Radio
                        id="scope-single-image"
                        name="scope-single-image"
                        isChecked={
                            values.scope.imageScope.registry === scopeContext.imageName.registry &&
                            values.scope.imageScope.remote === scopeContext.imageName.remote &&
                            values.scope.imageScope.tag === ALL
                        }
                        onChange={() =>
                            setFieldValue('scope.imageScope', {
                                ...scopeContext.imageName,
                                tag: ALL,
                            })
                        }
                        label={`All tags within ${scopeContext.imageName.registry}/${scopeContext.imageName.remote}`}
                    />
                    <Radio
                        id="scope-single-image-single-tag"
                        name="scope-single-image-single-tag"
                        isChecked={
                            values.scope.imageScope.registry === scopeContext.imageName.registry &&
                            values.scope.imageScope.remote === scopeContext.imageName.remote &&
                            values.scope.imageScope.tag === scopeContext.imageName.tag
                        }
                        onChange={() =>
                            setFieldValue('scope.imageScope', {
                                ...scopeContext.imageName,
                            })
                        }
                        label={`Only ${scopeContext.imageName.registry}/${scopeContext.imageName.remote}:${scopeContext.imageName.tag}`}
                    />
                </>
            )}
        </FormGroup>
    );
}

export default ExceptionScopeField;
