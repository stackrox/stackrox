import React from 'react';
import { Badge, Flex, List, ListItem, Title } from '@patternfly/react-core';

type ScanConfigProfilesViewProps = {
    headingLevel: 'h2' | 'h3';
    profiles: string[];
};

function ScanConfigProfilesView({
    headingLevel,
    profiles,
}: ScanConfigProfilesViewProps): React.ReactElement {
    return (
        <Flex direction={{ default: 'column' }}>
            <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                <Title headingLevel={headingLevel}>Profiles</Title>
                <Badge isRead>{profiles.length}</Badge>
            </Flex>
            <List isPlain>
                {profiles.map((profile) => (
                    <ListItem key={profile}>{profile}</ListItem>
                ))}
            </List>
        </Flex>
    );
}

export default ScanConfigProfilesView;
