import React, { useState } from 'react';
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
import { Select, SelectOption } from '@patternfly/react-core/deprecated';
import { TrashIcon } from '@patternfly/react-icons';
import { useField } from 'formik';

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
    const [isClusterSelectOpen, setIsClusterSelectOpen] = useState(false);
    const [isDeploymentSelectOpen, setIsDeploymentSelectOpen] = useState(false);
    const { value } = field;
    const { scope } = value || {};
    const { setValue } = helper;

    function handleChangeCluster(e, val) {
        setIsClusterSelectOpen(false);
        if (type === 'exclusion') {
            setValue({ ...value, scope: { ...scope, cluster: val } });
        } else {
            setValue({ ...value, cluster: val });
        }
    }

    function handleChangeDeployment(e, val) {
        setIsDeploymentSelectOpen(false);
        setValue({ ...value, name: val });
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
                                <Select
                                    onToggle={() => setIsClusterSelectOpen(!isClusterSelectOpen)}
                                    onSelect={handleChangeCluster}
                                    isOpen={isClusterSelectOpen}
                                    selections={value.cluster || scope?.cluster}
                                    placeholderText="Select a cluster"
                                    hasInlineFilter
                                    maxHeight="300px"
                                >
                                    {clusters.map((cluster) => (
                                        <SelectOption key={cluster.name} value={cluster.id}>
                                            {cluster.name}
                                        </SelectOption>
                                    ))}
                                </Select>
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
                                    <Select
                                        onToggle={() =>
                                            setIsDeploymentSelectOpen(!isDeploymentSelectOpen)
                                        }
                                        onSelect={handleChangeDeployment}
                                        isOpen={isDeploymentSelectOpen}
                                        selections={value.name}
                                        placeholderText="Select a deployment"
                                        isDisabled={hasAuditLogEventSource}
                                        hasInlineFilter
                                        maxHeight="300px"
                                    >
                                        {deployments.map((deployment) => (
                                            <SelectOption
                                                key={deployment.id}
                                                value={deployment.name}
                                            >
                                                {deployment.name}
                                            </SelectOption>
                                        ))}
                                    </Select>
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
