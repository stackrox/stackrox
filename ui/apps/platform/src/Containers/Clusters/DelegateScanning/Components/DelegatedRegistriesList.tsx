import React from 'react';
import { Button, FormGroup } from '@patternfly/react-core';
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
}: DelegatedRegistriesListProps) {
    return (
        <FormGroup label="Registries">
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
                        key="delegated-registries-table"
                    />
                </>
            ) : (
                <p>No registries specified.</p>
            )}
            {isEditing && (
                <Button
                    variant="link"
                    isInline
                    icon={<PlusCircleIcon />}
                    onClick={addRegistryRow}
                    className="pf-v5-u-mt-md"
                >
                    Add registry
                </Button>
            )}
        </FormGroup>
    );
}

export default DelegatedRegistriesList;
