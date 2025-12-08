import { Button } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { BaseImageReference } from 'services/BaseImagesService';
import { getTableUIState } from 'utils/getTableUIState';

import TBodyUnified from 'Components/TableStateTemplates/TbodyUnified';

export type BaseImagesTableProps = {
    baseImages: BaseImageReference[];
    onRemove: (baseImage: BaseImageReference) => void;
    isRemoveInProgress: boolean;
    isLoading: boolean;
    error: Error | null;
};

function BaseImagesTable({
    baseImages,
    onRemove,
    isRemoveInProgress,
    isLoading,
    error = null,
}: BaseImagesTableProps) {
    const tableState = getTableUIState({
        isLoading,
        data: baseImages,
        error: error || undefined,
        searchFilter: {},
    });

    return (
        <Table>
            <Thead>
                <Tr>
                    <Th>Repository Path</Th>
                    <Th>Tag Pattern</Th>
                    <Th width={10}>Actions</Th>
                </Tr>
            </Thead>
            <TBodyUnified<BaseImageReference>
                tableState={tableState}
                colSpan={3}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map((baseImage) => (
                            <Tr key={baseImage.id}>
                                <Td>{baseImage.baseImageRepoPath}</Td>
                                <Td>{baseImage.baseImageTagPattern}</Td>
                                <Td>
                                    {/* TODO: Add modal confirmation before removing */}
                                    <Button
                                        variant="secondary"
                                        isDanger
                                        isDisabled={isRemoveInProgress}
                                        onClick={() => onRemove(baseImage)}
                                    >
                                        Remove
                                    </Button>
                                </Td>
                            </Tr>
                        ))}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default BaseImagesTable;
