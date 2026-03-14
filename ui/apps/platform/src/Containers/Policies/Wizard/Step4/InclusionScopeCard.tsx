import { useState } from 'react';
import type { FormEvent, ReactElement } from 'react';
import { Flex, Form, FormGroup, Popover, Radio, TextInput } from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import TypeaheadSelect from 'Components/TypeaheadSelect/TypeaheadSelect';
import type { TypeaheadSelectOption } from 'Components/TypeaheadSelect/TypeaheadSelect';
import type { ClusterScopeObject } from 'services/RolesService';
import type { PolicyScope } from 'types/policy.proto';

import PolicyScopeCardBase from './PolicyScopeCardBase';

type Mode = 'name' | 'label';

type InclusionScopeCardProps = {
    scope: PolicyScope;
    index: number;
    clusters: ClusterScopeObject[];
    handleChange: (event: FormEvent<HTMLInputElement>, value: string) => void;
    setFieldValue: (field: string, value: unknown, shouldValidate?: boolean) => void;
    onDelete: () => void;
    hasAuditLogEventSource?: boolean;
};

function InclusionScopeCard({
    scope,
    index,
    clusters,
    handleChange,
    setFieldValue,
    onDelete,
    hasAuditLogEventSource = false,
}: InclusionScopeCardProps): ReactElement {
    const scopePath = `scope[${index}]`;
    const [clusterMode, setClusterMode] = useState<Mode>(scope.clusterLabel ? 'label' : 'name');
    const [namespaceMode, setNamespaceMode] = useState<Mode>(
        scope.namespaceLabel ? 'label' : 'name'
    );

    const clusterOptions: TypeaheadSelectOption[] = clusters.map((cluster) => ({
        value: cluster.id,
        label: cluster.name,
    }));

    function handleChangeNamespaceMode(mode: Mode) {
        setNamespaceMode(mode);

        if (mode === 'name') {
            setFieldValue(`${scopePath}.namespaceLabel`, null);
        } else {
            setFieldValue(`${scopePath}.namespace`, '');
        }
    }

    function handleChangeClusterMode(mode: Mode) {
        setClusterMode(mode);

        if (mode === 'name') {
            setFieldValue(`${scopePath}.clusterLabel`, null);
        } else {
            setFieldValue(`${scopePath}.cluster`, '');
        }
    }

    return (
        <PolicyScopeCardBase title="Scope" onDelete={onDelete}>
            <Form>
                <FormGroup label="Cluster" role="radiogroup">
                    <Flex direction={{ default: 'row' }}>
                        <Radio
                            id={`scope-${index}-cluster-by-name`}
                            name={`inclusion-scope-${index}-cluster-mode`}
                            label="By name"
                            isChecked={clusterMode === 'name'}
                            onChange={() => handleChangeClusterMode('name')}
                        />
                        <Radio
                            id={`scope-${index}-cluster-by-label`}
                            name={`inclusion-scope-${index}-cluster-mode`}
                            label="By label"
                            isChecked={clusterMode === 'label'}
                            onChange={() => handleChangeClusterMode('label')}
                        />
                    </Flex>
                    {clusterMode === 'name' ? (
                        <TypeaheadSelect
                            id={`${scopePath}-cluster`}
                            value={scope.cluster}
                            onChange={(clusterId) =>
                                setFieldValue(`${scopePath}.cluster`, clusterId)
                            }
                            options={clusterOptions}
                            placeholder="Select a cluster"
                            className="pf-v5-u-w-100"
                            isClearable
                        />
                    ) : (
                        <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                            <TextInput
                                aria-label="Cluster label key"
                                name={`${scopePath}.clusterLabel.key`}
                                onChange={handleChange}
                                placeholder="Label key"
                                type="text"
                                value={scope.clusterLabel?.key ?? ''}
                            />
                            <TextInput
                                aria-label="Cluster label value"
                                name={`${scopePath}.clusterLabel.value`}
                                onChange={handleChange}
                                placeholder="Label value"
                                type="text"
                                value={scope.clusterLabel?.value ?? ''}
                            />
                        </Flex>
                    )}
                </FormGroup>
                <FormGroup
                    label="Namespace"
                    role="radiogroup"
                    labelIcon={
                        <Popover
                            aria-label="Namespace help"
                            bodyContent="Use literals or regular expressions in RE2 syntax."
                        >
                            <button
                                type="button"
                                aria-label="More info for namespace field"
                                onClick={(e) => e.preventDefault()}
                                className="pf-v5-c-form__group-label-help"
                            >
                                <HelpIcon />
                            </button>
                        </Popover>
                    }
                >
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
                            name={`${scopePath}.namespace`}
                            onChange={handleChange}
                            placeholder="Namespace name"
                            type="text"
                            value={scope.namespace}
                        />
                    ) : (
                        <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                            <TextInput
                                aria-label="Namespace label key"
                                name={`${scopePath}.namespaceLabel.key`}
                                onChange={handleChange}
                                placeholder="Label key"
                                type="text"
                                value={scope.namespaceLabel?.key ?? ''}
                            />
                            <TextInput
                                aria-label="Namespace label value"
                                name={`${scopePath}.namespaceLabel.value`}
                                onChange={handleChange}
                                placeholder="Label value"
                                type="text"
                                value={scope.namespaceLabel?.value ?? ''}
                            />
                        </Flex>
                    )}
                </FormGroup>
                <FormGroup
                    label="Deployment"
                    labelIcon={
                        <Popover
                            aria-label="Deployment label help"
                            bodyContent="Use literals or regular expressions in RE2 syntax."
                        >
                            <button
                                type="button"
                                aria-label="More info for deployment label field"
                                onClick={(e) => e.preventDefault()}
                                className="pf-v5-c-form__group-label-help"
                            >
                                <HelpIcon />
                            </button>
                        </Popover>
                    }
                >
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput
                            aria-label="Deployment label key"
                            name={`${scopePath}.label.key`}
                            onChange={handleChange}
                            placeholder="Label key"
                            type="text"
                            value={scope.label?.key ?? ''}
                            isDisabled={hasAuditLogEventSource}
                        />
                        <TextInput
                            aria-label="Deployment label value"
                            name={`${scopePath}.label.value`}
                            onChange={handleChange}
                            placeholder="Label value"
                            type="text"
                            value={scope.label?.value ?? ''}
                            isDisabled={hasAuditLogEventSource}
                        />
                    </Flex>
                </FormGroup>
            </Form>
        </PolicyScopeCardBase>
    );
}

export default InclusionScopeCard;
