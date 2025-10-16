import type { ReactElement } from 'react';
import { Divider, FlexItem, Flex } from '@patternfly/react-core';

export type IntegrationFormActionsProps = {
    children: ReactElement | (ReactElement | null)[];
};

function IntegrationFormActions({ children }: IntegrationFormActionsProps): ReactElement {
    const integrationActionItems = React.Children.toArray(children).map((child, i) => {
        return (
            // eslint-disable-next-line react/no-array-index-key
            <FlexItem key={i} spacer={{ default: 'spacerMd' }}>
                {child}
            </FlexItem>
        );
    });

    return (
        <>
            <Divider component="div" />
            <Flex className="pf-v5-u-p-md">
                <FlexItem align={{ default: 'alignLeft' }}>
                    <Flex>{integrationActionItems}</Flex>
                </FlexItem>
            </Flex>
        </>
    );
}

export default IntegrationFormActions;
