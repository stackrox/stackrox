import React, { useState } from 'react';
import {
    Flex,
    FlexItem,
    Title,
    Text,
    LabelGroup,
    Label,
    Modal,
    ModalVariant,
    Button,
    Skeleton,
} from '@patternfly/react-core';
import { InfoCircleIcon } from '@patternfly/react-icons';

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
    const [isModalOpen, setIsModalOpen] = useState(false);

    if (isLoading) {
        return (
            <Flex
                className="pf-v5-u-p-lg pf-v5-u-background-color-100"
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
            <>
                <Flex
                    className="pf-v5-u-p-lg pf-v5-u-background-color-100"
                    direction={{ default: 'column' }}
                >
                    <Flex
                        justifyContent={{ default: 'justifyContentSpaceBetween' }}
                        alignItems={{ default: 'alignItemsFlexStart' }}
                    >
                        <FlexItem>
                            <Title headingLevel="h2">{profileName}</Title>
                        </FlexItem>
                        <FlexItem>
                            <LabelGroup numLabels={4}>
                                {profileVersion ? (
                                    <Label variant="filled">
                                        Profile version: {profileVersion}
                                    </Label>
                                ) : null}
                                <Label variant="filled">Applicability: {productType}</Label>
                                <Label
                                    color="blue"
                                    icon={<InfoCircleIcon />}
                                    onClick={() => setIsModalOpen(!isModalOpen)}
                                >
                                    View description
                                </Label>
                            </LabelGroup>
                        </FlexItem>
                    </Flex>
                    <FlexItem>
                        <Text className="pf-v5-u-font-size-sm">{title}</Text>
                    </FlexItem>
                </Flex>

                <Modal
                    variant={ModalVariant.medium}
                    title={profileName}
                    isOpen={isModalOpen}
                    onClose={() => setIsModalOpen(!isModalOpen)}
                    actions={[
                        <Button
                            key="close"
                            variant="primary"
                            onClick={() => setIsModalOpen(!isModalOpen)}
                        >
                            Close
                        </Button>,
                    ]}
                >
                    {description}
                </Modal>
            </>
        );
    }

    return null;
}

export default ProfileDetailsHeader;
