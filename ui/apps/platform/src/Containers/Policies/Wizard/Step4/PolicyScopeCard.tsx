import type { ReactElement } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    TextInput,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { useField } from 'formik';

import AutocompleteSelect from 'Components/CompoundSearchFilter/components/AutocompleteSelect';
import TypeaheadSelect from 'Components/TypeaheadSelect/TypeaheadSelect';
import type { TypeaheadSelectOption } from 'Components/TypeaheadSelect/TypeaheadSelect';
import type { ClusterScopeObject } from 'services/RolesService';
import type { SearchFilter } from 'types/search';

type PolicyScopeCardProps = {
    type: 'exclusion' | 'inclusion';
    name: string;
    clusters: ClusterScopeObject[];
    onDelete: () => void;
    hasAuditLogEventSource: boolean;
};

function PolicyScopeCard({
    type,
    name,
    clusters,
    onDelete,
    hasAuditLogEventSource = false,
}: PolicyScopeCardProps): ReactElement {
    const [field, , helper] = useField(name);
    const { value } = field;
    const { scope } = value ?? {};
    const { setValue } = helper;

    const clusterOptions: TypeaheadSelectOption[] = clusters.map((cluster) => ({
        value: cluster.id,
        label: cluster.name,
    }));

    // Note! Currently this filtering is only relevant to the exclusion scope, therefore accesses `value.scope` instead of `value`.
    // If at some point the inclusion scope gains a deployment filter, this will need to be updated.
    const selectedNamespaceValue = scope?.namespace;
    const deploymentSearchFilter: SearchFilter = {
        'Cluster ID': scope?.cluster ? [scope.cluster] : undefined,
        Namespace: selectedNamespaceValue ? [`r/${selectedNamespaceValue}`] : undefined,
    };

    function handleChangeCluster(selectedValue: string) {
        if (type === 'exclusion') {
            setValue({ ...value, scope: { ...scope, cluster: selectedValue } });
        } else {
            setValue({ ...value, cluster: selectedValue });
        }
    }

    function handleChangeDeployment(selectedValue: string) {
        const newValue = { ...value, name: selectedValue };
        // Do not pass an empty string to the backend, instead remove the field entirely
        if (!selectedValue) {
            delete newValue.name;
        }
        setValue(newValue);
    }

    function handleChangeLabelKey(key) {
        if (type === 'exclusion') {
            const { label } = scope ?? {};
            setValue({ ...value, scope: { ...scope, label: { ...label, key } } });
        } else {
            const { label } = value ?? {};
            setValue({ ...value, label: { ...label, key } });
        }
    }

    function handleChangeLabelValue(val) {
        if (type === 'exclusion') {
            const { label } = scope ?? {};
            setValue({ ...value, scope: { ...scope, label: { ...label, value: val } } });
        } else {
            const { label } = value ?? {};
            setValue({ ...value, label: { ...label, value: val } });
        }
    }

    function handleChangeNamespace(namespace) {
        if (type === 'exclusion') {
            setValue({ ...value, scope: { ...scope, namespace } });
        } else {
            setValue({ ...value, namespace });
        }
    }

    return (
        <Card>
            <CardHeader
                actions={{
                    actions: (
                        <>
                            <Divider orientation={{ default: 'vertical' }} component="div" />
                            <Button
                                icon={<TrashIcon />}
                                variant="plain"
                                className="pf-v6-u-mr-xs pf-v6-u-px-sm pf-v6-u-py-md"
                                onClick={onDelete}
                                title={`Delete ${type} scope`}
                            />
                        </>
                    ),
                    hasNoOffset: true,
                    className: undefined,
                }}
                className="pf-v6-u-p-0"
            >
                <CardTitle className="pf-v6-u-pl-lg">{type} scope</CardTitle>
            </CardHeader>
            <Divider component="div" />
            <CardBody>
                <Form>
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem>
                            <FormGroup label="Cluster" fieldId={`${name}-cluster`}>
                                <TypeaheadSelect
                                    id={`${name}-cluster`}
                                    value={value.cluster || scope?.cluster || ''}
                                    onChange={handleChangeCluster}
                                    options={clusterOptions}
                                    placeholder="Select a cluster"
                                    maxHeight="300px"
                                    className="pf-v6-u-w-100"
                                />
                            </FormGroup>
                        </FlexItem>
                        <FlexItem>
                            <FormGroup label="Namespace" fieldId={`${name}-namespace`}>
                                <TextInput
                                    value={value.namespace || scope?.namespace}
                                    type="text"
                                    id={`${name}-namespace`}
                                    onChange={(_event, namespace) =>
                                        handleChangeNamespace(namespace)
                                    }
                                    placeholder="Provide a namespace"
                                />
                            </FormGroup>
                        </FlexItem>
                        {type === 'exclusion' && (
                            <FlexItem>
                                <FormGroup label="Deployment" fieldId={`${name}-deployment`}>
                                    <AutocompleteSelect
                                        searchCategory="DEPLOYMENTS"
                                        searchTerm="Deployment"
                                        value={value.name || ''}
                                        onChange={handleChangeDeployment}
                                        onSearch={handleChangeDeployment}
                                        textLabel="Select a deployment"
                                        searchFilter={deploymentSearchFilter}
                                        isDisabled={hasAuditLogEventSource}
                                    />
                                </FormGroup>
                            </FlexItem>
                        )}
                        <FlexItem>
                            <FormGroup label="Deployment label" fieldId={`${name}-label`}>
                                <Flex
                                    direction={{ default: 'row' }}
                                    flexWrap={{ default: 'nowrap' }}
                                >
                                    <TextInput
                                        value={value.label?.key || scope?.label?.key}
                                        type="text"
                                        id={`${name}-label-key`}
                                        onChange={(_event, key) => handleChangeLabelKey(key)}
                                        placeholder="Label key"
                                        isDisabled={hasAuditLogEventSource}
                                    />
                                    <TextInput
                                        value={value.label?.value || scope?.label?.value}
                                        type="text"
                                        id={`${name}-label-value`}
                                        onChange={(_event, val) => handleChangeLabelValue(val)}
                                        placeholder="Label value"
                                        isDisabled={hasAuditLogEventSource}
                                    />
                                </Flex>
                            </FormGroup>
                        </FlexItem>
                    </Flex>
                </Form>
            </CardBody>
        </Card>
    );
}

export default PolicyScopeCard;
