import { Link } from 'react-router-dom-v5-compat';
import { Card, CardBody, CardTitle } from '@patternfly/react-core';

export type ContainerImageSummary = {
    id: string;
    idV2?: string;
    notPullable: boolean;
    name: { fullName: string } | null;
};

type ContainerImageInfoProps = {
    image: ContainerImageSummary | null;
    getImageUrl: (imageId: string) => string;
};

function ContainerImageInfo({ image, getImageUrl }: ContainerImageInfoProps) {
    if (!image) {
        return (
            <Card>
                <CardTitle>Image</CardTitle>
                <CardBody>
                    <em>image not available until deployment is running</em>
                </CardBody>
            </Card>
        );
    }

    const imageId = image.idV2 && image.idV2 !== '' ? image.idV2 : image.id;
    const imageName = image.name?.fullName || 'Image name not available';
    const imageDetailsPageURL = getImageUrl(imageId);

    if (imageId === '' || image.notPullable) {
        const unavailableText = image.notPullable
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
                <Link to={imageDetailsPageURL}>{imageName}</Link>
            </CardBody>
        </Card>
    );
}

export default ContainerImageInfo;
