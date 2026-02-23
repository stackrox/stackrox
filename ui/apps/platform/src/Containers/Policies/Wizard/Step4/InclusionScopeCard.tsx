import { useState } from 'react';
import type { FormEvent, ReactElement } from 'react';
import { Flex, Form, FormGroup, Radio, TextInput } from '@patternfly/react-core';

import type { PolicyScope, PolicyScopeLabel, ScopeLabelField } from 'types/policy.proto';

import PolicyScopeCardBase from './PolicyScopeCardBase';

type NamespaceMode = 'name' | 'label';

function getInitialNamespaceMode(scope: PolicyScope): NamespaceMode {
    return scope.namespaceLabel ? 'label' : 'name';
}

type InclusionScopeCardProps = {
    scope: PolicyScope;
    index: number;
    handleChange: (event: FormEvent<HTMLInputElement>, value: string) => void;
    onDelete: () => void;
};

function InclusionScopeCard({
    scope,
    index,
    handleChange,
    onDelete,
}: InclusionScopeCardProps): ReactElement {
    return (
        <PolicyScopeCardBase title="Inclusion scope" onDelete={onDelete}>
            <Form>
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
