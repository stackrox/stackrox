import React, { useState, ReactElement } from 'react';
import pluralize from 'pluralize';
import { Button, ButtonVariant, Flex, FlexItem, Label } from '@patternfly/react-core';

import entityTypes from 'constants/entityTypes';
import ImpactedEntitiesModal from './ImpactedEntitiesModal';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';

export type ImpactedEntitiesProps = {
    deployments: VulnerabilityRequest['deployments'];
    deploymentCount: VulnerabilityRequest['deploymentCount'];
    images: VulnerabilityRequest['images'];
    imageCount: VulnerabilityRequest['imageCount'];
};

function ImpactedEntities({
    deployments,
    deploymentCount,
    images,
    imageCount,
}: ImpactedEntitiesProps): ReactElement {
    const [modalTypeOpen, setModalTypeOpen] = useState('');

    function openModal(entityType) {
        setModalTypeOpen(entityType);
    }

    function closeModal() {
        setModalTypeOpen('');
    }
    return (
        <>
            <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                <FlexItem>
                    <Button
                        variant={ButtonVariant.link}
                        isInline
                        onClick={() => {
                            openModal(entityTypes.DEPLOYMENT);
                        }}
                    >
                        <Label color="blue">
                            {deploymentCount} {pluralize('deployment', deploymentCount)}
                        </Label>
                    </Button>
                </FlexItem>
                <FlexItem>
                    <Button
                        variant={ButtonVariant.link}
                        isInline
                        onClick={() => {
                            openModal(entityTypes.IMAGE);
                        }}
                    >
                        <Label color="blue">
                            {imageCount} {pluralize('image', imageCount)}
                        </Label>
                    </Button>
                </FlexItem>
            </Flex>
            <ImpactedEntitiesModal
                entityType={modalTypeOpen}
                isOpen={modalTypeOpen === entityTypes.DEPLOYMENT}
                entities={deployments}
                onClose={closeModal}
            />
            <ImpactedEntitiesModal
                entityType={modalTypeOpen}
                isOpen={modalTypeOpen === entityTypes.IMAGE}
                entities={images}
                onClose={closeModal}
            />
        </>
    );
}

export default ImpactedEntities;
