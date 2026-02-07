import { Content, Flex, PageSection, Title } from '@patternfly/react-core';

function CoveragesPageHeader() {
    return (
        <PageSection hasBodyWrapper={false} component="div">
            <Flex direction={{ default: 'column' }}>
                <Title headingLevel="h1">Coverage</Title>
                <Content component="p">
                    Assess profile compliance for nodes and platform resources across clusters
                </Content>
            </Flex>
        </PageSection>
    );
}

export default CoveragesPageHeader;
