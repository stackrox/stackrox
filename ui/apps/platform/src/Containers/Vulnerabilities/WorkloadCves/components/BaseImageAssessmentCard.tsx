import { Fragment, useState } from 'react';
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
    Stack,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

import { getDistanceStrict } from 'utils/dateUtils';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import type { BaseImageInfo } from './ImageDetailBadges';

export type BaseImageAssessmentCardProps = {
    baseImageInfo: BaseImageInfo[];
};

function BaseImageAssessmentCard({ baseImageInfo }: BaseImageAssessmentCardProps) {
    const [isExpanded, setIsExpanded] = useState(false);
    const { urlBuilder } = useWorkloadCveViewContext();

    const onToggle = (_event: ReactMouseEvent, expanded: boolean) => {
        setIsExpanded(expanded);
    };

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
                                <Fragment key={baseImage.baseImageId}>
                                    {index > 0 && <Divider />}
                                    <DescriptionList
                                        isCompact
                                        isHorizontal
                                        columnModifier={{ default: '1Col' }}
                                        termWidth="150px"
                                    >
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Image name</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                <Link to={imageDetailPath}>
                                                    {baseImage.baseImageFullName}
                                                </Link>
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
                                </Fragment>
                            );
                        })}
                    </Stack>
                </ExpandableSection>
            </CardBody>
        </Card>
    );
}

export default BaseImageAssessmentCard;
