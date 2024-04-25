import React from 'react';
import { PageSection, Flex, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

type TabNavSubHeaderProps = {
    description: string;
    actions: React.ReactNode;
};

function TabNavSubHeader({ description, actions }: TabNavSubHeaderProps) {
    return (
        <PageSection variant="light" className="pf-v5-u-py-0">
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem alignSelf="center">
                        <div className="pf-v5-u-font-size-sm">{description}</div>
                    </ToolbarItem>
                    <ToolbarItem align={{ default: 'alignRight' }}>
                        <Flex>{actions}</Flex>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
        </PageSection>
    );
}

export default TabNavSubHeader;
