import React from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import { ContainerResources } from 'types/deployment.proto';

type ContainerResourcesInfoProps = {
    resources: ContainerResources;
};

function ContainerResourcesInfo({ resources }: ContainerResourcesInfoProps) {
    return (
        <Card>
            <CardTitle>Resources</CardTitle>
            <CardBody className="pf-v5-u-background-color-200 pf-v5-u-pt-xl pf-v5-u-mx-lg pf-v5-u-mb-lg">
                <DescriptionList columnModifier={{ default: '2Col' }} isCompact>
                    <DescriptionListGroup>
                        <DescriptionListTerm>CPU requests (cores)</DescriptionListTerm>
                        <DescriptionListDescription>
                            {resources.cpuCoresRequest}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>CPU limet (cores)</DescriptionListTerm>
                        <DescriptionListDescription>
                            {resources.cpuCoresLimit}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Memory requests (MB)</DescriptionListTerm>
                        <DescriptionListDescription>
                            {resources.memoryMbRequest}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Memory limit (MB)</DescriptionListTerm>
                        <DescriptionListDescription>
                            {resources.memoryMbLimit}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </CardBody>
        </Card>
    );
}

export default ContainerResourcesInfo;
