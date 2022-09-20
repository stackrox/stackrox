import React from 'react';
import {
    PageSection,
    Title,
    Text,
    Button,
    Flex,
    FlexItem,
    ButtonVariant,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import LinkShim from 'Components/PatternFly/LinkShim';
import { collectionsPath } from 'routePaths';

type CollectionsTablePageProps = {
    hasWriteAccessForCollections: boolean;
};

function CollectionsTablePage({ hasWriteAccessForCollections }: CollectionsTablePageProps) {
    return (
        <>
            <PageTitle title="Collections" />
            <PageSection variant="light">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Collections</Title>
                        <Text>
                            Configure deployment collections to associate with other workflows
                        </Text>
                    </FlexItem>
                    {hasWriteAccessForCollections && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            <Button
                                variant={ButtonVariant.primary}
                                component={LinkShim}
                                href={`${collectionsPath}?action=create`}
                            >
                                Create collection
                            </Button>
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
        </>
    );
}

export default CollectionsTablePage;
