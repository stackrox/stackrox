import React from 'react';
import {
    Flex,
    FlexItem,
    Label,
    LabelGroup,
    LabelProps,
    Skeleton,
    Text,
    Title,
} from '@patternfly/react-core';

export type PageHeaderLabel = {
    text: string;
    icon: React.ReactNode;
    color: LabelProps['color'];
};

export type DetailsPageHeaderProps = {
    isLoading: boolean;
    name: string;
    labels?: PageHeaderLabel[];
    summary?: string;
    nameScreenReaderText: string;
    metadataScreenReaderText: string;
};

const MAX_NUM_LABELS = 7;

function DetailsPageHeader({
    isLoading,
    name,
    labels,
    summary,
    nameScreenReaderText,
    metadataScreenReaderText,
}: DetailsPageHeaderProps) {
    if (isLoading) {
        return (
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsXs' }}
                className="pf-u-w-50"
            >
                <Skeleton screenreaderText={nameScreenReaderText} fontSize="2xl" />
                <Skeleton screenreaderText={metadataScreenReaderText} height="100px" />
            </Flex>
        );
    }

    return (
        <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
            <FlexItem spacer={{ default: 'spacerMd' }}>
                <Title headingLevel="h1" className="pf-v5-u-mb-sm">
                    {name}
                </Title>
                {labels && labels.length !== 0 && (
                    <LabelGroup numLabels={MAX_NUM_LABELS}>
                        {labels.map((label) => {
                            return (
                                <Label variant="filled" icon={label.icon} color={label.color}>
                                    {label.text}
                                </Label>
                            );
                        })}
                    </LabelGroup>
                )}
            </FlexItem>
            {summary && (
                <FlexItem>
                    <Text>{summary}</Text>
                </FlexItem>
            )}
        </Flex>
    );
}

export default DetailsPageHeader;
