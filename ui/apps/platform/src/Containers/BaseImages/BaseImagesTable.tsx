import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { BaseImageReference } from 'services/BaseImagesService';
import { getTableUIState } from 'utils/getTableUIState';

import TBodyUnified from 'Components/TableStateTemplates/TbodyUnified';

export type BaseImagesTableProps = {
    baseImages: BaseImageReference[];
    hasWriteAccess: boolean;
    onEdit: (baseImage: BaseImageReference) => void;
    onDelete: (baseImage: BaseImageReference) => void;
    isActionInProgress: boolean;
    isLoading: boolean;
    error: Error | null;
};

function BaseImagesTable({
    baseImages,
    hasWriteAccess,
    onEdit,
    onDelete,
    isActionInProgress,
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
                    <Th>Base image path</Th>
                    <Th>Added by</Th>
                    {hasWriteAccess && (
                        <Th width={10}>
                            <span className="pf-v5-screen-reader">Row actions</span>
                        </Th>
                    )}
                </Tr>
            </Thead>
            <TBodyUnified<BaseImageReference>
                tableState={tableState}
                colSpan={hasWriteAccess ? 3 : 2}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map((baseImage) => (
                            <Tr key={baseImage.id}>
                                <Td>
                                    {baseImage.baseImageRepoPath}:{baseImage.baseImageTagPattern}
                                </Td>
                                <Td>{baseImage.user.name}</Td>
                                {hasWriteAccess && (
                                    <Td isActionCell>
                                        <ActionsColumn
                                            isDisabled={isActionInProgress}
                                            items={[
                                                {
                                                    title: 'Edit tag pattern',
                                                    onClick: () => onEdit(baseImage),
                                                },
                                                {
                                                    title: 'Delete base image',
                                                    onClick: () => onDelete(baseImage),
                                                },
                                            ]}
                                        />
                                    </Td>
                                )}
                            </Tr>
                        ))}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default BaseImagesTable;
