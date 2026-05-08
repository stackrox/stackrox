import { Link } from 'react-router-dom-v5-compat';
import { Card, CardBody, CardTitle } from '@patternfly/react-core';

import type { ContainerImage } from 'types/deployment.proto';

type ContainerImageInfoProps = {
    image: ContainerImage;
    getImageUrl: (imageId: string) => string;
};

function ContainerImageInfo({ image, getImageUrl }: ContainerImageInfoProps) {
    const imageId = image.idV2 && image.idV2 !== '' ? image.idV2 : image.id;
    const imageDetailsPageURL = getImageUrl(imageId);

    if (imageId === '' || image.notPullable) {
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
