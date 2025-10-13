import React, { ReactElement } from 'react';

import { PublicConfig } from 'types/config.proto';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Label,
} from '@patternfly/react-core';

export type PublicConfigLoginDetailsProps = {
    publicConfig: PublicConfig | null;
};

const PublicConfigLoginDetails = ({
    publicConfig,
}: PublicConfigLoginDetailsProps): ReactElement => {
    const isEnabled = publicConfig?.loginNotice?.enabled || false;
    const loginNoticeText = publicConfig?.loginNotice?.text || 'None';

    return (
        <Card isFlat data-testid="login-notice-config">
            <CardHeader
                actions={{
                    actions: (
                        <>
                            {isEnabled ? (
                                <Label color="green">Enabled</Label>
                            ) : (
                                <Label>Disabled</Label>
                            )}
                        </>
                    ),
                    hasNoOffset: false,
                    className: undefined,
                }}
                data-testid="login-notice-state"
            >
                {
                    <>
                        <CardTitle component="h3">Login configuration</CardTitle>
                    </>
                }
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

export default PublicConfigLoginDetails;
