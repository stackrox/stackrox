import React from 'react';
import {
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateActions,
    EmptyStateHeader,
    EmptyStateFooter,
    Button,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';

type BaseImagesEmptyStateProps = {
    onAddBaseImage: () => void;
};

function BaseImagesEmptyState({ onAddBaseImage }: BaseImagesEmptyStateProps) {
    return (
        <EmptyState variant="lg">
            <EmptyStateHeader
                titleText="Track base images to understand CVE sources"
                headingLevel="h2"
                icon={<EmptyStateIcon icon={CubesIcon} />}
            />
            <EmptyStateBody>
                Monitor CVEs in your base images and see which application images are affected
            </EmptyStateBody>
            <EmptyStateFooter>
                <EmptyStateActions>
                    <Button variant="primary" onClick={onAddBaseImage}>
                        Add your first base image
                    </Button>
                </EmptyStateActions>
            </EmptyStateFooter>
        </EmptyState>
    );
}

export default BaseImagesEmptyState;
