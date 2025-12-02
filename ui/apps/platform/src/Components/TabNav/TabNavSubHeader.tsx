import type { ReactElement, ReactNode } from 'react';
import { Flex, PageSection, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';

type TabNavSubHeaderProps = {
    description: string;
    actions: ReactNode;
};

function TabNavSubHeader({ description, actions }: TabNavSubHeaderProps): ReactElement {
    return (
        <PageSection hasBodyWrapper={false} className="pf-v6-u-py-0">
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem alignSelf="center">{description}</ToolbarItem>
                    <ToolbarItem align={{ default: 'alignEnd' }}>
                        <Flex>{actions}</Flex>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
        </PageSection>
    );
}

export default TabNavSubHeader;
