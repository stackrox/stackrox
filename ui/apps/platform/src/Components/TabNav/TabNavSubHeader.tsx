import React from 'react';
import type { ReactElement, ReactNode } from 'react';
import { PageSection, Flex, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

type TabNavSubHeaderProps = {
    description: string;
    actions: ReactNode;
};

function TabNavSubHeader({ description, actions }: TabNavSubHeaderProps): ReactElement {
    return (
        <PageSection variant="light" className="pf-v5-u-py-0">
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem alignSelf="center">{description}</ToolbarItem>
                    <ToolbarItem align={{ default: 'alignRight' }}>
                        <Flex>{actions}</Flex>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
        </PageSection>
    );
}

export default TabNavSubHeader;
