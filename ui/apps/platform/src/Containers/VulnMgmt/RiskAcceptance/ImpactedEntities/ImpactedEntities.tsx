import { Flex, FlexItem, Label } from '@patternfly/react-core';
import pluralize from 'pluralize';
import React, { ReactElement } from 'react';

export type DeferralExpirationDateProps = {
    deploymentCount: number;
    imageCount: number;
};

function ImpactedEntities({
    deploymentCount,
    imageCount,
}: DeferralExpirationDateProps): ReactElement {
    return (
        <Flex spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <Label color="blue">
                    {deploymentCount} {pluralize('deployment', deploymentCount)}
                </Label>
            </FlexItem>
            <FlexItem>
                <Label color="blue">
                    {imageCount} {pluralize('image', imageCount)}
                </Label>
            </FlexItem>
        </Flex>
    );
}

export default ImpactedEntities;
