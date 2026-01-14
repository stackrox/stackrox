import { useState } from 'react';
import type { MouseEvent as ReactMouseEvent } from 'react';
import {
    Card,
    CardBody,
    ClipboardCopy,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    ExpandableSection,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

// TODO: Uncomment when backend adds 'baseImageCreated' field to BaseImageInfo GraphQL type
// import { getDistanceStrict } from 'utils/dateUtils';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import type { BaseImageInfo } from './ImageDetailBadges';

export type BaseImageAssessmentCardProps = {
    baseImageInfo: BaseImageInfo[];
};

/**
 * Truncates a digest string for display, showing only the first 12 characters after the algorithm prefix.
 * Example: "sha256:abc123def456..." -> "sha256:abc123def456"
 */
function truncateDigest(digest: string): string {
    const parts = digest.split(':');
    if (parts.length === 2) {
        const [algorithm, hash] = parts;
        return `${algorithm}:${hash.slice(0, 12)}`;
    }
    return digest.slice(0, 19);
}

function BaseImageAssessmentCard({ baseImageInfo }: BaseImageAssessmentCardProps) {
    const [isExpanded, setIsExpanded] = useState(true);
    const { urlBuilder } = useWorkloadCveViewContext();

    const onToggle = (_event: ReactMouseEvent, expanded: boolean) => {
        setIsExpanded(expanded);
    };

    // Only render if there's at least one base image
    if (baseImageInfo.length === 0) {
        return null;
    }

    // For now, display the first detected base image
    // TODO: Handle multiple base images if needed in the future
    const baseImage = baseImageInfo[0];
    const imageDetailPath = urlBuilder.imageDetails(baseImage.baseImageId, 'OBSERVED');

    return (
        <Card isFlat isCompact>
            <CardBody>
                <ExpandableSection
                    toggleText="Base image assessment"
                    onToggle={onToggle}
                    isExpanded={isExpanded}
                >
                    <DescriptionList isCompact isHorizontal>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Detected base image</DescriptionListTerm>
                            <DescriptionListDescription>
                                <Flex
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                    alignItems={{ default: 'alignItemsCenter' }}
                                >
                                    <FlexItem>
                                        <Link to={imageDetailPath}>
                                            {baseImage.baseImageFullName}
                                        </Link>
                                    </FlexItem>
                                </Flex>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Digest</DescriptionListTerm>
                            <DescriptionListDescription>
                                <ClipboardCopy
                                    hoverTip="Copy digest"
                                    clickTip="Copied!"
                                    variant="inline-compact"
                                >
                                    {truncateDigest(baseImage.baseImageDigest)}
                                </ClipboardCopy>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        {/* TODO: Uncomment when backend adds 'baseImageCreated' field to BaseImageInfo GraphQL type
                        {baseImage.baseImageCreated && (
                            <DescriptionListGroup>
                                <DescriptionListTerm>Age</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {getDistanceStrict(baseImage.baseImageCreated, new Date())}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        )}
                        */}
                    </DescriptionList>
                </ExpandableSection>
            </CardBody>
        </Card>
    );
}

export default BaseImageAssessmentCard;
