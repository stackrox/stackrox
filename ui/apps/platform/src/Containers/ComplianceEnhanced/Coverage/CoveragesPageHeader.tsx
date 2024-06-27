import React from 'react';
import { Flex, Text, Title } from '@patternfly/react-core';

function CoveragesPageHeader() {
    return (
        <>
            <Flex
                className="pf-v5-u-p-lg"
                direction={{ default: 'column' }}
                justifyContent={{ default: 'justifyContentSpaceBetween' }}
            >
                <Title headingLevel="h1">Coverage</Title>
                <Text>
                    Assess profile compliance for nodes and platform resources across clusters
                </Text>
            </Flex>
        </>
    );
}

export default CoveragesPageHeader;
