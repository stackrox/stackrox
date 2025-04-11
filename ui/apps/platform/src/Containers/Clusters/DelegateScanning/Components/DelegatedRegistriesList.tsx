import React from 'react';
import { Button, FormGroup } from '@patternfly/react-core';
import PlusCircleIcon from '@patternfly/react-icons/dist/esm/icons/plus-circle-icon';

import {
    DelegatedRegistry,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';
import DelegatedRegistriesTable from './DelegatedRegistriesTable';

type DelegatedRegistriesListProps = {
    addRegistry: () => void;
    clusters: DelegatedRegistryCluster[];
    defaultClusterId: string;
    deleteRegistry: (indexToDelete: number) => void;
    isEditing: boolean;
    registries: DelegatedRegistry[];
    setRegistryClusterId: (indexToSet: number, clusterId: string) => void;
    setRegistryPath: (indexToSet: number, path: string) => void;
};

function DelegatedRegistriesList({
    addRegistry,
    clusters,
    defaultClusterId,
    deleteRegistry,
    isEditing,
    registries,
    setRegistryClusterId,
    setRegistryPath,
}: DelegatedRegistriesListProps) {
    return (
        <FormGroup label="Registries">
            {registries.length > 0 ? (
                <>
                    <DelegatedRegistriesTable
                        clusters={clusters}
                        defaultClusterId={defaultClusterId}
                        deleteRegistry={deleteRegistry}
                        isEditing={isEditing}
                        registries={registries}
                        setRegistryClusterId={setRegistryClusterId}
                        setRegistryPath={setRegistryPath}
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
                    onClick={addRegistry}
                    className="pf-v5-u-mt-md"
                >
                    Add registry
                </Button>
            )}
        </FormGroup>
    );
}

export default DelegatedRegistriesList;
