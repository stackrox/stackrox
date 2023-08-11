// TODO: remove lint override after @typescript-eslint deps can be resolved to ^5.2.x
/* eslint-disable react/prop-types */
import React from 'react';
import {
    Bullseye,
    Button,
    Card,
    CardFooter,
    EmptyState,
    EmptyStateBody,
    Title,
} from '@patternfly/react-core';
import PlusCircleIcon from '@patternfly/react-icons/dist/esm/icons/plus-circle-icon';

import {
    DelegatedRegistry,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';
import DelegatedRegistriesTable from './DelegatedRegistriesTable';

type DelegatedRegistriesListProps = {
    registries: DelegatedRegistry[];
    clusters: DelegatedRegistryCluster[];
    selectedClusterId: string;
    addRegistryRow: () => void;
    deleteRow: (number) => void;
    handlePathChange: (number, string) => void;
    handleClusterChange: (number, string) => void;
    // TODO: re-enable next type after @typescript-eslint deps can be resolved to ^5.2.x
    // updateRegistriesOrder: (DelegatedRegistry[]) => void;
};

function DelegatedRegistriesList({
    registries,
    clusters,
    selectedClusterId,
    handlePathChange,
    handleClusterChange,
    addRegistryRow,
    deleteRow,
    // TODO: remove lint override after @typescript-eslint deps can be resolved to ^5.2.x
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    updateRegistriesOrder,
}: DelegatedRegistriesListProps) {
    return (
        <Card className="pf-u-mb-lg">
            {registries.length > 0 ? (
                <>
                    <DelegatedRegistriesTable
                        registries={registries}
                        clusters={clusters}
                        selectedClusterId={selectedClusterId}
                        handlePathChange={handlePathChange}
                        handleClusterChange={handleClusterChange}
                        deleteRow={deleteRow}
                        // TODO: remove lint override after @typescript-eslint deps can be resolved to ^5.2.x
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        updateRegistriesOrder={updateRegistriesOrder}
                        key="delegated-registries-table"
                    />
                    <CardFooter>
                        <Button variant="link" icon={<PlusCircleIcon />} onClick={addRegistryRow}>
                            Add registry
                        </Button>
                    </CardFooter>
                </>
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
                        <Button variant="primary" onClick={addRegistryRow}>
                            Add registry
                        </Button>
                    </EmptyState>
                </Bullseye>
            )}
        </Card>
    );
}

export default DelegatedRegistriesList;
