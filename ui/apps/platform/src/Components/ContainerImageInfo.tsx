import { Link } from 'react-router-dom-v5-compat';
import { Card, CardBody, CardTitle } from '@patternfly/react-core';

import type { DeploymentContainerConfigData } from './DeploymentContainerConfig';

type ContainerImageInfoProps = {
    image: DeploymentContainerConfigData['image'];
    getImageUrl: (imageId: string) => string;
};

function ContainerImageInfo({ image, getImageUrl }: ContainerImageInfoProps) {
    const imageId = image?.idV2 && image.idV2 !== '' ? image.idV2 : (image?.id ?? '');
    const imageName = image?.name?.fullName || imageId || 'Image unavailable';

    if (imageId === '' || image?.notPullable) {
        const unavailableText = image?.notPullable
            ? 'image not currently pullable'
            : 'image not available until deployment is running';
        return (
            <Card>
                <CardTitle>Image</CardTitle>
                <CardBody>
                    <span>{imageName}</span> <em>({unavailableText})</em>
                </CardBody>
            </Card>
        );
    }

    return (
        <Card>
            <CardTitle>Image</CardTitle>
            <CardBody>
                <Link to={getImageUrl(imageId)}>{imageName}</Link>
            </CardBody>
        </Card>
    );
}

export default ContainerImageInfo;
