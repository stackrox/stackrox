import type { ReactElement, ReactNode } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardFooter,
    CardHeader,
    CardTitle,
    CodeBlock,
    CodeBlockCode,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import type { PolicyScope } from 'types/policy.proto';

type PolicyScopeCardBaseProps = {
    title: string;
    onDelete: () => void;
    children: ReactNode;
    scope: PolicyScope | undefined;
    clusterName?: string;
};

function PolicyScopeCardBase({
    title,
    onDelete,
    scope,
    clusterName,
    children,
}: PolicyScopeCardBaseProps): ReactElement {
    let clusterDisplay = 'Cluster: all';
    if (scope?.clusterLabel?.key) {
        const { key, value } = scope.clusterLabel;
        clusterDisplay = `Cluster label: ${key}${value ? `=${value}` : ''}`;
    } else if (scope?.cluster) {
        clusterDisplay = `Cluster: ${clusterName ?? scope.cluster}`;
    }

    let namespaceDisplay = 'Namespace: all';
    if (scope?.namespaceLabel?.key) {
        const { key, value } = scope.namespaceLabel;
        namespaceDisplay = `Namespace label: ${key}${value ? `=${value}` : ''}`;
    } else if (scope?.namespace) {
        namespaceDisplay = `Namespace: ${scope.namespace}`;
    }

    let deploymentDisplay = 'Deployment: all';
    if (scope?.label) {
        const { label } = scope;
        deploymentDisplay = `Deployment label: ${label.key}${label.value ? `=${label.value}` : ''}`;
    }

    return (
        <Card isCompact>
            <CardHeader
                actions={{
                    actions: (
                        <Button variant="plain" onClick={onDelete} title={`Delete ${title}`}>
                            <TrashIcon />
                        </Button>
                    ),
                }}
            >
                <CardTitle>{title}</CardTitle>
            </CardHeader>
            <CardBody>{children}</CardBody>
            <CardFooter>
                Applies to:
                <CodeBlock>
                    <CodeBlockCode>
                        {[
                            `(${clusterDisplay})`,
                            `AND (${namespaceDisplay})`,
                            `AND (${deploymentDisplay})`,
                        ].join('\n')}
                    </CodeBlockCode>
                </CodeBlock>
            </CardFooter>
        </Card>
    );
}

export default PolicyScopeCardBase;
