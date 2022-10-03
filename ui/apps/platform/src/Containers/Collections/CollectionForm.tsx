import React from 'react';
import { Card, CardBody, Flex, FlexItem, Title } from '@patternfly/react-core';

import { ResolvedCollectionResponse } from 'services/CollectionsService';
import CollectionResults from './CollectionResults';
import { CollectionPageAction } from './collections.utils';
import RuleSelector from './RuleSelector';
import Ipsum from './Ipsum_DELETE';

export type CollectionFormProps = {
    action: CollectionPageAction['type'];
    collectionData: ResolvedCollectionResponse | undefined;
};

function CollectionForm({ action, collectionData }: CollectionFormProps) {
    return (
        <Flex alignItems={{ default: 'alignItemsStretch' }}>
            <FlexItem style={{ flexBasis: 0 }} flex={{ default: 'flex_2' }}>
                <Flex
                    direction={{ default: 'column' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    className="pf-u-h-100"
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
                            <Ipsum />
                        </CardBody>
                    </Card>
                    <Card>
                        <CardBody>
                            <Title headingLevel="h2">Attach existing collections</Title>
                            <Ipsum />
                        </CardBody>
                    </Card>
                </Flex>
            </FlexItem>
            <FlexItem
                style={{
                    position: 'sticky',
                    top: 'var(--pf-c-page__main-section--PaddingTop)',
                    maxHeight: 'var(--collection-results-container-max-height, 100%)',
                }}
                flex={{ default: 'flex_1' }}
            >
                <Card className="pf-u-h-100" style={{ overflow: 'auto' }}>
                    <CardBody>
                        <CollectionResults />
                        <Ipsum />
                    </CardBody>
                </Card>
            </FlexItem>
        </Flex>
    );
}

export default CollectionForm;
