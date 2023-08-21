import React, { useState } from 'react';
import {
    Card,
    CardHeader,
    CardTitle,
    CardActions,
    CardBody,
    Divider,
    Button,
    Select,
    SelectOption,
    TextInput,
    Flex,
    FlexItem,
    Form,
    FormGroup,
} from '@patternfly/react-core';
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
            <CardHeader className="pf-u-p-0">
                <CardTitle className="pf-u-pl-lg">{type} scope</CardTitle>
                <CardActions hasNoOffset>
                    <Divider isVertical component="div" />
                    <Button
                        variant="plain"
                        className="pf-u-mr-xs pf-u-px-sm pf-u-py-md"
                        onClick={onDelete}
                    >
                        <TrashIcon />
                    </Button>
                </CardActions>
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
                                    onChange={handleChangeNamespace}
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
                            <FormGroup label="Label" fieldId={`${name}-label`}>
                                <Flex
                                    direction={{ default: 'row' }}
                                    flexWrap={{ default: 'nowrap' }}
                                >
                                    <TextInput
                                        value={value.label?.key || scope?.label?.key}
                                        type="text"
                                        id={`${name}-label-key`}
                                        onChange={handleChangeLabelKey}
                                        placeholder="Label key"
                                        isDisabled={hasAuditLogEventSource}
                                    />
                                    <TextInput
                                        value={value.label?.value || scope?.label?.value}
                                        type="text"
                                        id={`${name}-label-value`}
                                        onChange={handleChangeLabelValue}
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
