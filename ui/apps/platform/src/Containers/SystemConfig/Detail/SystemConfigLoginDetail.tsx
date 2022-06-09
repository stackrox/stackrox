import React, { ReactElement } from 'react';

import { PublicConfig } from 'types/config.proto';
import {
    Card,
    CardActions,
    CardBody,
    CardHeader,
    CardHeaderMain,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Label,
} from '@patternfly/react-core';

export type SystemConfigLoginDetailProps = {
    publicConfig: PublicConfig | null;
};

const SystemConfigLoginDetail = ({ publicConfig }: SystemConfigLoginDetailProps): ReactElement => {
    const isEnabled = publicConfig?.loginNotice?.enabled || false;
    const loginNoticeText = publicConfig?.loginNotice?.text || 'None';

    return (
        <Card data-testid="login-notice-config">
            <CardHeader>
                <CardHeaderMain>
                    <CardTitle>Login configuration</CardTitle>
                </CardHeaderMain>
                <CardActions data-testid="login-notice-state">
                    {isEnabled ? <Label color="green">Enabled</Label> : <Label>Disabled</Label>}
                </CardActions>
            </CardHeader>
            <Divider component="div" />
            <CardBody>
                <DescriptionList>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Text (2000 character limit)</DescriptionListTerm>
                        <DescriptionListDescription>{loginNoticeText}</DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </CardBody>
        </Card>
    );
};

export default SystemConfigLoginDetail;
