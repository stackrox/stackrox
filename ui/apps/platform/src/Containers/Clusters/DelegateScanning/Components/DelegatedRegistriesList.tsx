import React from 'react';
import {
    Bullseye,
    Button,
    Card,
    CardBody,
    EmptyState,
    EmptyStateBody,
    Title,
} from '@patternfly/react-core';

import { DelegatedRegistry } from 'services/DelegatedRegistryConfigService';

type DelegatedRegistriesListProps = {
    registries: DelegatedRegistry[];
};

function DelegatedRegistriesList({ registries }: DelegatedRegistriesListProps) {
    return (
        <Card className="pf-u-mb-lg">
            {registries.length > 0 ? (
                <CardBody>
                    <p>(table goes here)</p>
                </CardBody>
            ) : (
                <Bullseye className="pf-u-flex-grow-1">
                    <EmptyState>
                        <Title headingLevel="h2" size="lg">
                            No registries specified.
                        </Title>
                        <EmptyStateBody>
                            <p>All scans will be delegated to the default cluster.</p>
                            <p>You can override this for specific registries.</p>
                        </EmptyStateBody>
                        <Button variant="primary">Add registry</Button>
                    </EmptyState>
                </Bullseye>
            )}
        </Card>
    );
}

export default DelegatedRegistriesList;
