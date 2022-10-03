import React from 'react';
import { Card, CardBody, CardTitle, Flex, FlexItem, Title } from '@patternfly/react-core';

import { ResolvedCollectionResponse } from 'services/CollectionsService';
import CollectionResults from './CollectionResults';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';

export type CollectionFormProps = {
    action: CollectionPageAction['type'];
    collectionData: ResolvedCollectionResponse | undefined;
};

function CollectionForm({ action, collectionData }: CollectionFormProps) {
    return (
        <Flex alignItems={{ default: 'alignItemsStretch' }}>
            <FlexItem grow={{ default: 'grow' }}>
                <Flex
                    direction={{ default: 'column' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    <Card>
                        <CardBody>
                            <Title headingLevel="h2">Collection details</Title>
                        </CardBody>
                    </Card>
                    <Card>
                        <CardBody>
                            <Title headingLevel="h2">Add new collection rules</Title>
                            <RuleSelector />
                            <RuleSelector />
                            <RuleSelector />
                        </CardBody>
                    </Card>
                    <Card>
                        <CardBody>
                            <Title headingLevel="h2">Attach existing collections</Title>
                        </CardBody>
                    </Card>
                </Flex>
            </FlexItem>
            <FlexItem>
                <Card>
                    <CardBody>
                        <CollectionResults />
                    </CardBody>
                </Card>
            </FlexItem>
        </Flex>
    );
}

export default CollectionForm;
