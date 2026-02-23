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
    onChange: (newScope: PolicyScope) => void;
    onDelete: () => void;
};

function InclusionScopeCard({ scope, onChange, onDelete }: InclusionScopeCardProps): ReactElement {
    const [namespaceMode, setNamespaceMode] = useState<NamespaceMode>(() =>
        getInitialNamespaceMode(scope)
    );

    function handleChangeScopeLabel(
        labelField: ScopeLabelField,
        labelKey: keyof PolicyScopeLabel,
        labelValue: string
    ) {
        const current = scope[labelField];
        const updated: PolicyScopeLabel = {
            key: current?.key ?? '',
            value: current?.value ?? '',
            [labelKey]: labelValue,
        };
        onChange({ ...scope, [labelField]: updated });
    }

    function handleChangeNamespaceMode(mode: NamespaceMode) {
        setNamespaceMode(mode);
        if (mode === 'name') {
            onChange({ ...scope, namespaceLabel: null });
        } else {
            onChange({ ...scope, namespace: '' });
        }
    }

    function handleChangeNamespace(_event: FormEvent, namespace: string) {
        onChange({ ...scope, namespace });
    }

    return (
        <PolicyScopeCardBase title="Inclusion scope" onDelete={onDelete}>
            <Form>
                <FormGroup label="Namespace" role="radiogroup">
                    <Flex direction={{ default: 'row' }}>
                        <Radio
                            id={`${scope.cluster}-namespace-by-name`}
                            name={`${scope.cluster}-namespace-mode`}
                            label="By name"
                            isChecked={namespaceMode === 'name'}
                            onChange={() => handleChangeNamespaceMode('name')}
                        />
                        <Radio
                            id={`${scope.cluster}-namespace-by-label`}
                            name={`${scope.cluster}-namespace-mode`}
                            label="By label"
                            isChecked={namespaceMode === 'label'}
                            onChange={() => handleChangeNamespaceMode('label')}
                        />
                    </Flex>
                    {namespaceMode === 'name' ? (
                        <TextInput
                            aria-label="Namespace name"
                            onChange={handleChangeNamespace}
                            placeholder="Namespace name"
                            type="text"
                            value={scope.namespace}
                        />
                    ) : (
                        <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                            <TextInput
                                aria-label="Namespace label key"
                                onChange={(_e, key) =>
                                    handleChangeScopeLabel('namespaceLabel', 'key', key)
                                }
                                placeholder="Label key"
                                type="text"
                                value={scope.namespaceLabel?.key ?? ''}
                            />
                            <TextInput
                                aria-label="Namespace label value"
                                onChange={(_e, val) =>
                                    handleChangeScopeLabel('namespaceLabel', 'value', val)
                                }
                                placeholder="Label value"
                                type="text"
                                value={scope.namespaceLabel?.value ?? ''}
                            />
                        </Flex>
                    )}
                </FormGroup>
                <FormGroup label="Deployment label">
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput
                            aria-label="Deployment label key"
                            onChange={(_e, key) => handleChangeScopeLabel('label', 'key', key)}
                            placeholder="Label key"
                            type="text"
                            value={scope.label?.key ?? ''}
                        />
                        <TextInput
                            aria-label="Deployment label value"
                            onChange={(_e, value) =>
                                handleChangeScopeLabel('label', 'value', value)
                            }
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
