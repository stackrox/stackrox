import React from 'react';
import { CodeBlock, Flex, CodeBlockCode } from '@patternfly/react-core';
import { TableDataRow } from '../Tables/table.utils';

export type DockerfileLayerProps = {
    layer: TableDataRow['layer'];
};

function DockerfileLayer({ layer }: DockerfileLayerProps) {
    return layer ? (
        <CodeBlock>
            <Flex>
                <CodeBlockCode className="pf-u-flex-nowrap">
                    {layer.line} {layer.instruction}
                </CodeBlockCode>
                <CodeBlockCode className="pf-u-flex-grow-1 pf-u-flex-basis-0">
                    {layer.value}
                </CodeBlockCode>
            </Flex>
        </CodeBlock>
    ) : (
        <CodeBlock>
            <CodeBlockCode>Dockerfile layer information not available</CodeBlockCode>
        </CodeBlock>
    );
}

export default DockerfileLayer;
