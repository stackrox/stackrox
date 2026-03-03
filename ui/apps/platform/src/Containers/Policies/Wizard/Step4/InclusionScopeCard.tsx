import { useState } from 'react';
import type { FormEvent, ReactElement } from 'react';
import { Flex, Form, FormGroup, Radio, TextInput } from '@patternfly/react-core';

import type { PolicyScope } from 'types/policy.proto';

import PolicyScopeCardBase from './PolicyScopeCardBase';

type NamespaceMode = 'name' | 'label';

type InclusionScopeCardProps = {
    scope: PolicyScope;
    index: number;
    handleChange: (event: FormEvent<HTMLInputElement>, value: string) => void;
    setFieldValue: (field: string, value: unknown, shouldValidate?: boolean) => void;
    onDelete: () => void;
};

function InclusionScopeCard({
    scope,
    index,
    handleChange,
    setFieldValue,
    onDelete,
}: InclusionScopeCardProps): ReactElement {
    const [namespaceMode, setNamespaceMode] = useState<NamespaceMode>(
        scope.namespaceLabel ? 'label' : 'name'
    );

    function handleChangeNamespaceMode(mode: NamespaceMode) {
        setNamespaceMode(mode);

        if (mode === 'name') {
            setFieldValue(`scope[${index}].namespaceLabel`, null);
        } else {
            setFieldValue(`scope[${index}].namespace`, '');
        }
    }

    return (
        <PolicyScopeCardBase title="Inclusion scope" onDelete={onDelete}>
            <Form>
                <FormGroup label="Namespace" role="radiogroup">
                    <Flex direction={{ default: 'row' }}>
                        <Radio
                            id={`scope-${index}-namespace-by-name`}
                            name={`inclusion-scope-${index}-namespace-mode`}
                            label="By name"
                            isChecked={namespaceMode === 'name'}
                            onChange={() => handleChangeNamespaceMode('name')}
                        />
                        <Radio
                            id={`scope-${index}-namespace-by-label`}
                            name={`inclusion-scope-${index}-namespace-mode`}
                            label="By label"
                            isChecked={namespaceMode === 'label'}
                            onChange={() => handleChangeNamespaceMode('label')}
                        />
                    </Flex>
                    {namespaceMode === 'name' ? (
                        <TextInput
                            aria-label="Namespace name"
                            name={`scope[${index}].namespace`}
                            onChange={handleChange}
                            placeholder="Namespace name"
                            type="text"
                            value={scope.namespace}
                        />
                    ) : (
                        <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                            <TextInput
                                aria-label="Namespace label key"
                                name={`scope[${index}].namespaceLabel.key`}
                                onChange={handleChange}
                                placeholder="Label key"
                                type="text"
                                value={scope.namespaceLabel?.key ?? ''}
                            />
                            <TextInput
                                aria-label="Namespace label value"
                                name={`scope[${index}].namespaceLabel.value`}
                                onChange={handleChange}
                                placeholder="Label value"
                                type="text"
                                value={scope.namespaceLabel?.value ?? ''}
                            />
                        </Flex>
                    )}
                </FormGroup>
                <FormGroup label="Deployment">
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput
                            aria-label="Deployment label key"
                            name={`scope[${index}].label.key`}
                            onChange={handleChange}
                            placeholder="Label key"
                            type="text"
                            value={scope.label?.key ?? ''}
                        />
                        <TextInput
                            aria-label="Deployment label value"
                            name={`scope[${index}].label.value`}
                            onChange={handleChange}
                            placeholder="Label value"
                            type="text"
                            value={scope.label?.value ?? ''}
                        />
                    </Flex>
                </FormGroup>
            </Form>
        </PolicyScopeCardBase>
    );
}

export default InclusionScopeCard;
