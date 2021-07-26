import React, { ReactElement } from 'react';
import { Divider, FlexItem, Flex } from '@patternfly/react-core';

export type IntegrationFormToolBarProps = {
    children: ReactElement | ReactElement[];
};

function IntegrationFormToolBar({ children }: IntegrationFormToolBarProps): ReactElement {
    const integrationToolBarItems = React.Children.toArray(children).map((child) => {
        return <FlexItem spacer={{ default: 'spacerMd' }}>{child}</FlexItem>;
    });

    return (
        <>
            <Flex className="pf-u-p-md">
                <FlexItem align={{ default: 'alignRight' }}>
                    <Flex>{integrationToolBarItems}</Flex>
                </FlexItem>
            </Flex>
            <Divider component="div" />
        </>
    );
}

export default IntegrationFormToolBar;
