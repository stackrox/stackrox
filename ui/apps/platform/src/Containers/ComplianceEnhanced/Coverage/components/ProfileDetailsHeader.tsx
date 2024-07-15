import React from 'react';
import {
    ExpandableSectionVariant,
    ExpandableSection,
    Flex,
    FlexItem,
    Label,
    LabelGroup,
    Text,
    Title,
    Skeleton,
} from '@patternfly/react-core';

import { ComplianceProfileSummary } from 'services/ComplianceCommon';

interface ProfileDetailsHeaderProps {
    isLoading: boolean;
    profileDetails: ComplianceProfileSummary | undefined;
    profileName: string;
}

function ProfileDetailsHeader({
    isLoading,
    profileDetails,
    profileName,
}: ProfileDetailsHeaderProps) {
    const [isExpanded, setIsExpanded] = React.useState(false);

    function onToggleDescription(_event: React.MouseEvent, isExpanded: boolean) {
        setIsExpanded(isExpanded);
    }

    if (isLoading) {
        return (
            <Flex
                className="pf-v5-u-p-md pf-v5-u-background-color-100"
                direction={{ default: 'column' }}
            >
                <Title headingLevel="h2">{profileName}</Title>
                <Skeleton screenreaderText="Loading profile details" />
            </Flex>
        );
    }

    if (profileDetails) {
        const { description, productType, profileVersion, title } = profileDetails;

        return (
            <Flex
                className="pf-v5-u-p-md pf-v5-u-background-color-100"
                direction={{ default: 'column' }}
            >
                <Flex
                    alignItems={{ default: 'alignItemsFlexStart' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    <FlexItem>
                        <Title headingLevel="h2">{profileName}</Title>
                    </FlexItem>
                    <FlexItem>
                        <LabelGroup numLabels={4}>
                            {profileVersion ? (
                                <Label variant="filled">Profile version: {profileVersion}</Label>
                            ) : null}
                            <Label variant="filled">Applicability: {productType}</Label>
                        </LabelGroup>
                    </FlexItem>
                </Flex>
                <FlexItem>
                    <Text className="pf-v5-u-font-size-sm">{title}</Text>
                </FlexItem>
                <FlexItem>
                    <ExpandableSection
                        className="pf-v5-u-font-size-sm"
                        isExpanded={isExpanded}
                        toggleText={isExpanded ? 'Show less' : 'Show more'}
                        truncateMaxLines={5}
                        variant={ExpandableSectionVariant.truncate}
                        onToggle={onToggleDescription}
                    >
                        {description}
                    </ExpandableSection>
                </FlexItem>
            </Flex>
        );
    }

    return null;
}

export default ProfileDetailsHeader;
