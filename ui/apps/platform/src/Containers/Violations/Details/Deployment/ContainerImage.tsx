import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';

import { vulnManagementPath } from 'routePaths';
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
    if (image.id === '' || image.notPullable) {
        const unavailableText = image.notPullable
            ? 'image not currently pullable'
            : 'image not available until deployment is running';
        const NameComponent = (
            <div>
                {image.name.fullName}
                <span className="pf-u-pl-sm">({unavailableText})</span>
            </div>
        );
        return <DescriptionListItem term="Image name" desc={NameComponent} />;
    }
    return (
        <DescriptionListItem
            term="Image name"
            desc={<Link to={`${vulnManagementPath}/image/${image.id}`}>{image.name.fullName}</Link>}
        />
    );
}

export default ContainerImage;
