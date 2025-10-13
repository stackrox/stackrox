import React from 'react';
import {
    Card,
    CardHeader,
    CardTitle,
    CardBody,
    Divider,
    Button,
    TextInput,
    Flex,
    FlexItem,
    Form,
    FormGroup,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { useField } from 'formik';

import TypeaheadSelect, { TypeaheadSelectOption } from 'Components/TypeaheadSelect/TypeaheadSelect';
import { ClusterScopeObject } from 'services/RolesService';
import { ListDeployment } from 'types/deployment.proto';

type PolicyScopeCardProps = {
    type: 'exclusion' | 'inclusion';
    name: string;
    clusters: ClusterScopeObject[];
    deployments?: ListDeployment[];
    onDelete: () => void;
    hasAuditLogEventSource: boolean;
};

function PolicyScopeCard({
    type,
    name,
    clusters,
    deployments = [],
    onDelete,
    hasAuditLogEventSource = false,
}: PolicyScopeCardProps): React.ReactElement {
    const [field, , helper] = useField(name);
    const { value } = field;
    const { scope } = value || {};
    const { setValue } = helper;

    const clusterOptions: TypeaheadSelectOption[] = clusters.map((cluster) => ({
        value: cluster.id,
        label: cluster.name,
    }));

    const deploymentOptions: TypeaheadSelectOption[] = deployments.map((deployment) => ({
        value: deployment.name,
        label: deployment.name,
    }));

    function handleChangeCluster(selectedValue: string) {
        if (type === 'exclusion') {
            setValue({ ...value, scope: { ...scope, cluster: selectedValue } });
        } else {
            setValue({ ...value, cluster: selectedValue });
        }
    }

    function handleChangeDeployment(selectedValue: string) {
        setValue({ ...value, name: selectedValue });
    }

    function handleChangeLabelKey(key) {
        if (type === 'exclusion') {
            const { label } = scope || {};
            setValue({ ...value, scope: { ...scope, label: { ...label, key } } });
        } else {
            const { label } = value || {};
            setValue({ ...value, label: { ...label, key } });
        }
    }

    function handleChangeLabelValue(val) {
        if (type === 'exclusion') {
            const { label } = scope || {};
            setValue({ ...value, scope: { ...scope, label: { ...label, value: val } } });
        } else {
            const { label } = value || {};
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
                                variant="plain"
                                className="pf-v5-u-mr-xs pf-v5-u-px-sm pf-v5-u-py-md"
                                onClick={onDelete}
                                title={`Delete ${type} scope`}
                            >
                                <TrashIcon />
                            </Button>
                        </>
                    ),
                    hasNoOffset: true,
                    className: undefined,
                }}
                className="pf-v5-u-p-0"
            >
                <CardTitle className="pf-v5-u-pl-lg">{type} scope</CardTitle>
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
                                    className="pf-v5-u-w-100"
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
                                    <TypeaheadSelect
                                        id={`${name}-deployment`}
                                        value={value.name || ''}
                                        onChange={handleChangeDeployment}
                                        options={deploymentOptions}
                                        placeholder="Select a deployment"
                                        isDisabled={hasAuditLogEventSource}
                                        maxHeight="300px"
                                        className="pf-v5-u-w-100"
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
