import React from 'react';
import { Card, CardBody, CardTitle, List, ListItem } from '@patternfly/react-core';

type ScanConfigProfilesProps = {
    profiles: string[];
};

function ScanConfigProfiles({ profiles }: ScanConfigProfilesProps): React.ReactElement {
    return (
        <Card className="pf-u-h-100">
            <CardTitle component="h2">Profiles</CardTitle>
            <CardBody>
                <List isPlain>
                    {profiles.map((profile) => (
                        <ListItem key={profile}>{profile}</ListItem>
                    ))}
                </List>
            </CardBody>
        </Card>
    );
}

export default ScanConfigProfiles;
