import React from 'react';
import { PageSection, Flex, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

type PolicyManagementSubHeaderProps = {
    description: string;
    actions: React.ReactNode;
};

function PolicyManagementSubHeader({ description, actions }: PolicyManagementSubHeaderProps) {
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

export default PolicyManagementSubHeader;
