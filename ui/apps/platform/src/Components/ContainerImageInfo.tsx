import React from 'react';
import { Link } from 'react-router-dom';
import { Card, CardBody, CardTitle } from '@patternfly/react-core';

import { vulnManagementPath, vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { ContainerImage } from 'types/deployment.proto';
import useFeatureFlags from 'hooks/useFeatureFlags';

type ContainerImageInfoProps = {
    image: ContainerImage; // note: the k8s API, and our data of it, use singular "command" for this array
};

function ContainerImageInfo({ image }: ContainerImageInfoProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const areVMMiscImprovementsEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_2_MISC_IMPROVEMENTS');

    const imageDetailsPageURL = areVMMiscImprovementsEnabled
        ? `${vulnerabilitiesWorkloadCvesPath}/images/${image.id}`
        : `${vulnManagementPath}/image/${image.id}`;

    if (image.id === '' || image.notPullable) {
        const unavailableText = image.notPullable
            ? 'image not currently pullable'
            : 'image not available until deployment is running';
        return (
            <Card>
                <CardTitle>Image</CardTitle>
                <CardBody>
                    <span>{image.name.fullName}</span> <em>({unavailableText})</em>
                </CardBody>
            </Card>
        );
    }

    return (
        <Card>
            <CardTitle>Image</CardTitle>
            <CardBody>
                <Link to={imageDetailsPageURL}>{image.name.fullName}</Link>
            </CardBody>
        </Card>
    );
}

export default ContainerImageInfo;
