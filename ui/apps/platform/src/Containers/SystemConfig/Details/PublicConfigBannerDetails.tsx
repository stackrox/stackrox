import type { ReactElement } from 'react';
import capitalize from 'lodash/capitalize';
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

import ColorPicker from 'Components/ColorPicker';
import type { PublicConfig } from 'types/config.proto';

type BannerType = 'header' | 'footer';

export type PublicConfigBannerDetailsProps = {
    type: BannerType;
    publicConfig: PublicConfig | null;
};

const PublicConfigBannerDetails = ({
    type,
    publicConfig,
}: PublicConfigBannerDetailsProps): ReactElement => {
    const {
        backgroundColor = null,
        color = null,
        enabled = false,
        size = 'None',
        text = 'None',
    } = publicConfig?.[type] || {};

    const title = `${capitalize(type)} configuration`;

    return (
        <Card isFlat data-testid={`${type}-config`}>
            <CardHeader
                actions={{
                    actions: (
                        <>
                            {enabled ? (
                                <Label color="green">Enabled</Label>
                            ) : (
                                <Label>Disabled</Label>
                            )}
                        </>
                    ),
                    hasNoOffset: false,
                    className: undefined,
                }}
                data-testid={`${type}-state`}
            >
                {
                    <>
                        <CardTitle component="h3">{title}</CardTitle>
                    </>
                }
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
                            <ColorPicker
                                id={`publicConfig.${type}.color`}
                                label={`Text color of ${type}`}
                                color={color}
                                disabled
                            />
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
                            <ColorPicker
                                id={`publicConfig.${type}.backgroundColor`}
                                label={`Background color of ${type}`}
                                color={backgroundColor}
                                disabled
                            />
                            {backgroundColor || 'None'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                </DescriptionList>
            </CardBody>
        </Card>
    );
};

export default PublicConfigBannerDetails;
