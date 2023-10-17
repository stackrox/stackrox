import React from 'react';
import { FormGroup, FormGroupProps, Radio } from '@patternfly/react-core';

import { ScopeContext } from './utils';

export type ExceptionScopeFieldProps = {
    fieldId: FormGroupProps['fieldId'];
    label: FormGroupProps['label'];
    scopeContext: ScopeContext;
};

function ExceptionScopeField({ fieldId, label, scopeContext }: ExceptionScopeFieldProps) {
    return (
        <FormGroup fieldId={fieldId} label={label} isRequired>
            {scopeContext === 'GLOBAL' && (
                <Radio
                    id="scope-global"
                    name="scope-global"
                    isDisabled
                    isChecked
                    label="Selected CVEs across all images and deployments"
                />
            )}
            {scopeContext !== 'GLOBAL' && (
                <>
                    <Radio
                        id="scope-single-image"
                        name="scope-single-image"
                        isChecked={false}
                        onChange={() => {}}
                        label={`All tags within ${scopeContext.image.name}}`}
                    />
                    <Radio
                        id="scope-single-image-single-tag"
                        name="scope-single-image-single-tag"
                        isChecked={false}
                        onChange={() => {}}
                        label={`Only ${scopeContext.image.name}:${scopeContext.image.tag}`}
                    />
                </>
            )}
        </FormGroup>
    );
}

export default ExceptionScopeField;
