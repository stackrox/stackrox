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
    Divider,
    ExpandableSection,
    Flex,
    FlexItem,
    Stack,
    StackItem,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

import { getDistanceStrict } from 'utils/dateUtils';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import type { BaseImageInfo } from './ImageDetailBadges';

export type BaseImageAssessmentCardProps = {
    baseImageInfo: BaseImageInfo[];
};

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

    return (
        <Card isFlat isCompact>
            <CardBody>
                <ExpandableSection
                    toggleText="Base image assessment"
                    onToggle={onToggle}
                    isExpanded={isExpanded}
                >
                    <Stack hasGutter>
                        {baseImageInfo.map((baseImage, index) => {
                            const imageDetailPath = urlBuilder.imageDetails(
                                baseImage.baseImageId,
                                'OBSERVED'
                            );
                            return (
                                <StackItem key={baseImage.baseImageId}>
                                    {index > 0 && <Divider className="pf-v5-u-mb-md" />}
                                    <DescriptionList
                                        isCompact
                                        isHorizontal
                                        columnModifier={{ default: '1Col' }}
                                        termWidth="150px"
                                    >
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Image name</DescriptionListTerm>
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
                                            <DescriptionListTerm>Image digest</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                <ClipboardCopy
                                                    hoverTip="Copy digest"
                                                    clickTip="Copied!"
                                                    variant="inline-compact"
                                                >
                                                    {baseImage.baseImageDigest}
                                                </ClipboardCopy>
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                        {baseImage.baseImageCreated && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>Image age</DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {getDistanceStrict(
                                                        baseImage.baseImageCreated,
                                                        new Date()
                                                    )}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                    </DescriptionList>
                                </StackItem>
                            );
                        })}
                    </Stack>
                </ExpandableSection>
            </CardBody>
        </Card>
    );
}

export default BaseImageAssessmentCard;
