import React from 'react';
import { CodeBlock, Flex, CodeBlockCode } from '@patternfly/react-core';
import { TableDataRow } from '../Tables/table.utils';

export type DockerfileLayerTdProps = {
    layer: TableDataRow['layer'];
};

function DockerfileLayerTd({ layer }: DockerfileLayerTdProps) {
    return layer ? (
        <CodeBlock>
            <Flex>
                <CodeBlockCode
                    // 120px is a width that looks good with the largest dockerfile instruction: "HEALTHCHECK"
                    style={{ flexBasis: '120px' }}
                    className="pf-u-flex-shrink-0"
                >
                    {layer.line} {layer.instruction}
                </CodeBlockCode>
                <CodeBlockCode className="pf-u-flex-grow-1 pf-u-flex-basis-0">
                    {layer.instruction} {layer.value}
                </CodeBlockCode>
            </Flex>
        </CodeBlock>
    ) : (
        <CodeBlock>
            <CodeBlockCode>Dockerfile layer information not available</CodeBlockCode>
        </CodeBlock>
    );
}

export default DockerfileLayerTd;
