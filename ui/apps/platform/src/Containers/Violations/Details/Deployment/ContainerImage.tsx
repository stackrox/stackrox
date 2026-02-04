import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';

import DescriptionListItem from 'Components/DescriptionListItem';
import type { ContainerImage as ContainerImageType } from 'types/deployment.proto';

type ContainerImageProps = {
    image: ContainerImageType;
    vulnMgmtBasePath: string;
};

function ContainerImage({ image, vulnMgmtBasePath }: ContainerImageProps): ReactElement {
    const imageId = image.idV2 && image.idV2 !== '' ? image.idV2 : image.id;
    const imageDetailsPageURL = `${vulnMgmtBasePath}/images/${imageId}`;

    if (imageId === '' || image.notPullable) {
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
