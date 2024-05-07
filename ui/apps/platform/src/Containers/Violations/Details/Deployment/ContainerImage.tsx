import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';

import { vulnManagementPath, vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import useFeatureFlags from 'hooks/useFeatureFlags';

import DescriptionListItem from 'Components/DescriptionListItem';

type ContainerImageProps = {
    image: {
        name: {
            fullName: string;
            registry: string;
            remote: string;
            tag: string;
        };
        notPullable: boolean;
        id: string;
    };
};

function ContainerImage({ image }: ContainerImageProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const areVMMiscImprovementsEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_2_MISC_IMPROVEMENTS');

    const imageDetailsPageURL = areVMMiscImprovementsEnabled
        ? `${vulnerabilitiesWorkloadCvesPath}/images/${image.id}`
        : `${vulnManagementPath}/image/${image.id}`;

    if (image.id === '' || image.notPullable) {
        const unavailableText = image.notPullable
            ? 'image not currently pullable'
            : 'image not available until deployment is running';
        const NameComponent = (
            <div>
                {image.name.fullName}
                <span className="pf-v5-u-pl-sm">({unavailableText})</span>
            </div>
        );
        return <DescriptionListItem term="Image name" desc={NameComponent} />;
    }
    return (
        <DescriptionListItem
            term="Image name"
            desc={<Link to={imageDetailsPageURL}>{image.name.fullName}</Link>}
        />
    );
}

export default ContainerImage;
