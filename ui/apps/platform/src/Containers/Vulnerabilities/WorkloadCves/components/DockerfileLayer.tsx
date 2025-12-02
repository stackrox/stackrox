import { CodeBlock, CodeBlockCode, Flex } from '@patternfly/react-core';
import type { TableDataRow } from '../Tables/table.utils';

export type DockerfileLayerProps = {
    layer: TableDataRow['layer'];
};

function DockerfileLayer({ layer }: DockerfileLayerProps) {
    return layer ? (
        <CodeBlock>
            <Flex>
                <CodeBlockCode className="pf-v6-u-flex-nowrap">
                    {layer.line} {layer.instruction}
                </CodeBlockCode>
                <CodeBlockCode className="pf-v6-u-flex-grow-1 pf-v6-u-flex-basis-0">
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
