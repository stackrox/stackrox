import type { FormEvent, ReactElement } from 'react';
import { Flex, Form, FormGroup, TextInput } from '@patternfly/react-core';

import TypeaheadSelect from 'Components/TypeaheadSelect/TypeaheadSelect';
import type { TypeaheadSelectOption } from 'Components/TypeaheadSelect/TypeaheadSelect';
import type { ClusterScopeObject } from 'services/RolesService';
import type { PolicyExcludedDeployment } from 'types/policy.proto';

import PolicyScopeCardBase from './PolicyScopeCardBase';

type ExclusionScopeCardProps = {
    excludedDeploymentScope: PolicyExcludedDeployment;
    index: number;
    clusters: ClusterScopeObject[];
    handleChange: (event: FormEvent<HTMLInputElement>, value: string) => void;
    setFieldValue: (field: string, value: unknown, shouldValidate?: boolean) => void;
    onDelete: () => void;
};

function ExclusionScopeCard({
    excludedDeploymentScope,
    index,
    clusters,
    handleChange,
    setFieldValue,
    onDelete,
}: ExclusionScopeCardProps): ReactElement {
    const scopePath = `excludedDeploymentScopes[${index}]`;

    const clusterOptions: TypeaheadSelectOption[] = clusters.map((cluster) => ({
        value: cluster.id,
        label: cluster.name,
    }));

    return (
        <PolicyScopeCardBase title="Exclusion" onDelete={onDelete}>
            <Form>
                <FormGroup label="Cluster">
                    <Flex direction={{ default: 'column' }}>
                        <TypeaheadSelect
                            id={`${scopePath}-cluster`}
                            value={excludedDeploymentScope.scope?.cluster ?? ''}
                            onChange={(clusterId) =>
                                setFieldValue(`${scopePath}.scope.cluster`, clusterId)
                            }
                            options={clusterOptions}
                            placeholder="Select a cluster"
                            isClearable
                        />
                    </Flex>
                </FormGroup>
                <FormGroup label="Namespace">
                    <TextInput
                        aria-label="Namespace name"
                        name={`${scopePath}.scope.namespace`}
                        onChange={handleChange}
                        placeholder="Namespace name"
                        type="text"
                        value={excludedDeploymentScope.scope?.namespace ?? ''}
                    />
                </FormGroup>
                <FormGroup label="Deployment">
                    <TextInput
                        aria-label="Deployment name"
                        name={`${scopePath}.name`}
                        onChange={handleChange}
                        placeholder="Deployment name"
                        type="text"
                        value={excludedDeploymentScope.name ?? ''}
                    />
                </FormGroup>
                <FormGroup label="Deployment label">
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput
                            aria-label="Deployment label key"
                            name={`${scopePath}.scope.label.key`}
                            onChange={handleChange}
                            placeholder="Label key"
                            type="text"
                            value={excludedDeploymentScope.scope?.label?.key ?? ''}
                        />
                        <TextInput
                            aria-label="Deployment label value"
                            name={`${scopePath}.scope.label.value`}
                            onChange={handleChange}
                            placeholder="Label value"
                            type="text"
                            value={excludedDeploymentScope.scope?.label?.value ?? ''}
                        />
                    </Flex>
                </FormGroup>
            </Form>
        </PolicyScopeCardBase>
    );
}

export default ExclusionScopeCard;
