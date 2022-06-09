import React, { ReactElement } from 'react';

import capitalize from 'lodash/capitalize';
import ColorPicker from 'Components/ColorPicker';
import { PublicConfig } from 'types/config.proto';
import {
    Card,
    CardActions,
    CardTitle,
    CardBody,
    CardHeader,
    CardHeaderMain,
    Label,
    DescriptionList,
    DescriptionListGroup,
    DescriptionListTerm,
    DescriptionListDescription,
    Divider,
} from '@patternfly/react-core';

type BannerType = 'header' | 'footer';

export type SystemConfigBannerDetailProps = {
    type: BannerType;
    publicConfig: PublicConfig | null;
};

const SystemConfigBannerDetail = ({
    type,
    publicConfig,
}: SystemConfigBannerDetailProps): ReactElement => {
    const {
        backgroundColor = null,
        color = null,
        enabled = false,
        size = 'None',
        text = 'None',
    } = publicConfig?.[type] || {};

    const title = `${capitalize(type)} configuration`;

    return (
        <Card data-testid={`${type}-config`}>
            <CardHeader>
                <CardHeaderMain>
                    <CardTitle>{title}</CardTitle>
                </CardHeaderMain>
                <CardActions data-testid={`${type}-state`}>
                    {enabled ? <Label color="green">Enabled</Label> : <Label>Disabled</Label>}
                </CardActions>
            </CardHeader>
            <Divider component="div" />
            <CardBody>
                <DescriptionList
                    columnModifier={{
                        default: '2Col',
                    }}
                >
                    <DescriptionListGroup>
                        <DescriptionListTerm>Text (2000 character limit)</DescriptionListTerm>
                        <DescriptionListDescription>{text}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Text color</DescriptionListTerm>
                        <DescriptionListDescription>
                            <ColorPicker color={color} disabled />
                            {color || 'None'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Text size</DescriptionListTerm>
                        <DescriptionListDescription>{capitalize(size)}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Background color</DescriptionListTerm>
                        <DescriptionListDescription>
                            <ColorPicker color={backgroundColor} disabled />
                            {backgroundColor || 'None'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </CardBody>
        </Card>
    );
};

export default SystemConfigBannerDetail;
