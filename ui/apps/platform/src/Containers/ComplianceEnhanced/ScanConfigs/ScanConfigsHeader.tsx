import React from 'react';
import {
    Flex,
    PageSection,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';

type ScanConfigsHeaderProps = {
    description: string;
    actions: React.ReactNode;
};

function ScanConfigsHeader({ description, actions }: ScanConfigsHeaderProps) {
    return (
        <>
            <PageTitle title="Scan Schedules" />
            <PageSection variant="light">
                <Title headingLevel="h1">Scan schedules</Title>
            </PageSection>
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
        </>
    );
}

export default ScanConfigsHeader;
