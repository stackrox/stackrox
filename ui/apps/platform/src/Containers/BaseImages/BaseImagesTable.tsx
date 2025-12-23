import { ActionsColumn, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { BaseImageReference } from 'services/BaseImagesService';
import { getTableUIState } from 'utils/getTableUIState';

import TBodyUnified from 'Components/TableStateTemplates/TbodyUnified';

export type BaseImagesTableProps = {
    baseImages: BaseImageReference[];
    onEdit: (baseImage: BaseImageReference) => void;
    onRemove: (baseImage: BaseImageReference) => void;
    isActionInProgress: boolean;
    isLoading: boolean;
    error: Error | null;
};

function BaseImagesTable({
    baseImages,
    onEdit,
    onRemove,
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
                    <Th width={10}>
                        <span className="pf-v5-screen-reader">Row actions</span>
                    </Th>
                </Tr>
            </Thead>
            <TBodyUnified<BaseImageReference>
                tableState={tableState}
                colSpan={3}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map((baseImage) => (
                            <Tr key={baseImage.id}>
                                <Td>
                                    {baseImage.baseImageRepoPath}:{baseImage.baseImageTagPattern}
                                </Td>
                                <Td>{baseImage.user.name}</Td>
                                <Td isActionCell>
                                    <ActionsColumn
                                        isDisabled={isActionInProgress}
                                        items={[
                                            {
                                                title: 'Edit',
                                                onClick: () => onEdit(baseImage),
                                            },
                                            {
                                                title: 'Remove',
                                                onClick: () => onRemove(baseImage),
                                            },
                                        ]}
                                    />
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
