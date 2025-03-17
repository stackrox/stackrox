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
    EmptyStateHeader,
    EmptyStateFooter,
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
    isEditing: boolean;
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
    isEditing,
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
        <Card className="pf-v5-u-mb-lg">
            {registries.length > 0 ? (
                <>
                    <DelegatedRegistriesTable
                        registries={registries}
                        clusters={clusters}
                        selectedClusterId={selectedClusterId}
                        isEditing={isEditing}
                        handlePathChange={handlePathChange}
                        handleClusterChange={handleClusterChange}
                        deleteRow={deleteRow}
                        // TODO: remove lint override after @typescript-eslint deps can be resolved to ^5.2.x
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        updateRegistriesOrder={updateRegistriesOrder}
                        key="delegated-registries-table"
                    />
                    {isEditing && (
                        <CardFooter>
                            <Button
                                variant="link"
                                icon={<PlusCircleIcon />}
                                onClick={addRegistryRow}
                            >
                                Add registry
                            </Button>
                        </CardFooter>
                    )}
                </>
            ) : (
                <Bullseye className="pf-v5-u-flex-grow-1">
                    <EmptyState>
                        <EmptyStateHeader titleText="No registries specified." headingLevel="h2" />
                        <EmptyStateBody>
                            <p>All scans will be delegated to the default cluster.</p>
                            <p>You can override this for specific registries.</p>
                        </EmptyStateBody>
                        {isEditing && (
                            <EmptyStateFooter>
                                <Button variant="primary" onClick={addRegistryRow}>
                                    Add registry
                                </Button>
                            </EmptyStateFooter>
                        )}
                    </EmptyState>
                </Bullseye>
            )}
        </Card>
    );
}

export default DelegatedRegistriesList;
