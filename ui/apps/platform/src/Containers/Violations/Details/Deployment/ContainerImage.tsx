import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';

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
    vulnMgmtBasePath: string;
};

function ContainerImage({ image, vulnMgmtBasePath }: ContainerImageProps): ReactElement {
    const imageDetailsPageURL = `${vulnMgmtBasePath}/images/${image.id}`;

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
