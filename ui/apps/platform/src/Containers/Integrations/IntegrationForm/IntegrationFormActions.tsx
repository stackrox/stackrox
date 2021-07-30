import React, { ReactElement } from 'react';
import { Divider, FlexItem, Flex } from '@patternfly/react-core';

export type IntegrationFormActionsProps = {
    children: ReactElement | ReactElement[];
};

function IntegrationFormActions({ children }: IntegrationFormActionsProps): ReactElement {
    const integrationActionItems = React.Children.toArray(children).map((child) => {
        return <FlexItem spacer={{ default: 'spacerMd' }}>{child}</FlexItem>;
    });

    return (
        <>
            <Divider component="div" />
            <Flex className="pf-u-p-md">
                <FlexItem align={{ default: 'alignLeft' }}>
                    <Flex>{integrationActionItems}</Flex>
                </FlexItem>
            </Flex>
        </>
    );
}

export default IntegrationFormActions;
