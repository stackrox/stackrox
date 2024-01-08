import React from 'react';
import { PageSection, Flex, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

type TabNavSubHeaderProps = {
    description: string;
    actions: React.ReactNode;
};

function TabNavSubHeader({ description, actions }: TabNavSubHeaderProps) {
    return (
        <PageSection variant="light" className="pf-u-py-0">
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <div className="pf-u-font-size-sm">{description}</div>
                    </ToolbarItem>
                    <ToolbarItem alignment={{ default: 'alignRight' }}>
                        <Flex>{actions}</Flex>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
        </PageSection>
    );
}

export default TabNavSubHeader;
