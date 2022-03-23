import React from 'react';
import { Flex, FlexItem, Form, FormGroup } from '@patternfly/react-core';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import NamespaceSelect from './Header/NamespaceSelect';
import ClusterSelect from './Header/ClusterSelect';

const clusterSelectId = 'noSelectedNamespace.clusterSelect';
const namespaceSelectId = 'noSelectedNamespace.namespaceSelect';

function NoSelectedNamespace() {
    return (
        <Flex className="pf-u-flex-grow-1 pf-u-pt-2xl">
            <FlexItem grow={{ default: 'grow' }}>
                <EmptyStateTemplate
                    headingLevel="h2"
                    title="Please select at least one namespace from your cluster"
                >
                    <Form className="pf-u-px-2xl pf-u-py-md pf-u-display-flex pf-u-flex-direction-row pf-u-text-align-left">
                        <FormGroup
                            className="pf-u-w-50"
                            label="Cluster"
                            isRequired
                            fieldId={clusterSelectId}
                        >
                            <ClusterSelect id={clusterSelectId} />
                        </FormGroup>
                        <FormGroup
                            className="pf-u-w-50"
                            label="Namespace(s)"
                            isRequired
                            fieldId={namespaceSelectId}
                        >
                            <NamespaceSelect id={namespaceSelectId} />
                        </FormGroup>
                    </Form>
                </EmptyStateTemplate>
            </FlexItem>
        </Flex>
    );
}

export default NoSelectedNamespace;
