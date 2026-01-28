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
    Label,
    LabelGroup,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom-v5-compat';

import { getDistanceStrict } from 'utils/dateUtils';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import type { BaseImage } from './ImageDetailBadges';

export type BaseImageAssessmentCardProps = {
    baseImage: BaseImage;
};

function BaseImageAssessmentCard({ baseImage }: BaseImageAssessmentCardProps) {
    const [isExpanded, setIsExpanded] = useState(false);
    const { urlBuilder } = useWorkloadCveViewContext();

    const onToggle = (_event: ReactMouseEvent, expanded: boolean) => {
        setIsExpanded(expanded);
    };

    // Use the digest (imageSha) as the image ID for the detail link
    const imageDetailPath = urlBuilder.imageDetails(baseImage.imageSha, 'OBSERVED');

    return (
        <Card isFlat isCompact>
            <CardBody>
                <ExpandableSection
                    toggleText="Base image assessment"
                    onToggle={onToggle}
                    isExpanded={isExpanded}
                >
                    <DescriptionList
                        isCompact
                        isHorizontal
                        columnModifier={{ default: '1Col' }}
                        termWidth="150px"
                    >
                        <DescriptionListGroup>
                            <DescriptionListTerm>
                                {baseImage.names.length > 1 ? 'Image names' : 'Image name'}
                            </DescriptionListTerm>
                            <DescriptionListDescription>
                                <LabelGroup numLabels={3} isCompact>
                                    {baseImage.names.map((name) => (
                                        <Label
                                            key={name}
                                            color="blue"
                                            isCompact
                                            render={({ className, content }) => (
                                                <Link to={imageDetailPath} className={className}>
                                                    {content}
                                                </Link>
                                            )}
                                        >
                                            {name}
                                        </Label>
                                    ))}
                                </LabelGroup>
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
                                    {baseImage.imageSha}
                                </ClipboardCopy>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        {baseImage.created && (
                            <DescriptionListGroup>
                                <DescriptionListTerm>Image age</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {getDistanceStrict(baseImage.created, new Date())}
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        )}
                    </DescriptionList>
                </ExpandableSection>
            </CardBody>
        </Card>
    );
}

export default BaseImageAssessmentCard;
