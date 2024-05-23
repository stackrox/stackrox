import React, { useState } from 'react';
import {
    CodeBlock,
    CodeBlockCode,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { compoundSearchFilter } from 'Components/CompoundSearchFilter/types';

function DemoPage() {
    const [history, setHistory] = useState<string[]>([]);
    const historyHeader = '//// Search History ////';
    const historyText = `\n\n${history.join('\n')}`;
    const content = `${historyHeader}${history.length !== 0 ? historyText : ''}`;
    return (
        <>
            <PageTitle title="Demo - Advanced Filters" />
            <PageSection variant="light">
                <Flex>
                    <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Demo - Advanced Filters</Title>
                        <FlexItem>
                            This section will demo the capabilities of advanced filters. NOT A REAL
                            PAGE
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <PageSection variant="light">
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsLg' }}
                    >
                        <CompoundSearchFilter
                            config={compoundSearchFilter}
                            onSearch={(value) => {
                                setHistory((prevState: string[]) => {
                                    return [...prevState, value];
                                });
                            }}
                        />
                        <CodeBlock>
                            <CodeBlockCode id="code-content">{content}</CodeBlockCode>{' '}
                        </CodeBlock>
                    </Flex>
                </PageSection>
            </PageSection>
        </>
    );
}

export default DemoPage;
