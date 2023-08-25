import React from 'react';
import { Link } from 'react-router-dom';
import { Card, CardBody, CardTitle } from '@patternfly/react-core';

import { vulnManagementPath } from 'routePaths';
import { ContainerImage } from 'types/deployment.proto';

type ContainerImageInfoProps = {
    image: ContainerImage; // note: the k8s API, and our data of it, use singular "command" for this array
};

function ContainerImageInfo({ image }: ContainerImageInfoProps) {
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
                <Link to={`${vulnManagementPath}/image/${image.id}`}>{image.name.fullName}</Link>
            </CardBody>
        </Card>
    );
}

export default ContainerImageInfo;
