import React, { useEffect, useState } from 'react';
import {
    Button,
    ButtonVariant,
    Modal,
    ModalBoxBody,
    ModalBoxFooter,
    ModalVariant,
    PageSection,
    TextInput,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import resolvePath from 'object-resolve-path';
import LinkShim from '../../../../../Components/PatternFly/LinkShim';
import { integrationsPath } from '../../../../../routePaths';
import tableColumnDescriptor from '../../../../Integrations/utils/tableColumnDescriptor';
import {
    fetchSignatureIntegrations,
    SignatureIntegration,
} from '../../../../../services/SignatureIntegrationsService';
import useTableSelection from '../../../../../hooks/useTableSelection';

type TableCellProps = {
    row: SignatureIntegration;
    column: {
        Header: string;
        accessor: ((data) => string) | string;
    };
};

function TableCellValue({ row, column }: TableCellProps): React.ReactElement {
    let value: string;
    if (typeof column.accessor === 'function') {
        value = column.accessor(row).toString();
    } else {
        value = resolvePath(row, column.accessor).toString();
    }
    return <div>{value || '-'}</div>;
}

export function ImageSignersCriteriaFieldInput({ name, setValue, value }): React.ReactElement {
    const [isModalOpen, setIsModalOpen] = React.useState(false);

    function handleCancelModal() {
        setIsModalOpen(false);
    }

    function handleOpenModal() {
        setIsModalOpen(true);
    }

    const columns = [...tableColumnDescriptor.signatureIntegrations.signature];
    const [integrations, setIntegrations] = useState<SignatureIntegration[]>([]);

    useEffect(() => {
        fetchSignatureIntegrations()
            .then((data) => {
                setIntegrations(data);
            })
            .catch(() => {
                setIntegrations([]);
            });
    }, []);

    const { selected, onSelect, onSelectAll, allRowsSelected, getSelectedIds } =
        // pass an array with pre-selected values([true, false, true])
        useTableSelection<SignatureIntegration>(integrations);

    // enabled if selection changed
    function onConfirmHandler() {
        setValue({ arrayValue: getSelectedIds() });
        setIsModalOpen(false);
    }

    return (
        <>
            <TextInput
                id={name}
                isDisabled
                value={
                    value.arrayValue?.length === 0
                        ? 'Select trusted image signers'
                        : `Selected ${value.arrayValue.length} trusted image signers`
                }
            />
            <Button key="open" variant={ButtonVariant.primary} onClick={handleOpenModal}>
                Select
            </Button>
            <Modal
                title="Select trusted image signers"
                isOpen={isModalOpen}
                variant={ModalVariant.small}
                onClose={handleCancelModal}
                data-testid="select-image-signers"
                aria-label="Select image signers"
                hasNoBodyWrapper
            >
                <ModalBoxBody>
                    <PageSection variant="light">
                        <TableComposable variant="compact" isStickyHeader>
                            <Thead>
                                <Tr>
                                    <Th
                                        select={{
                                            onSelect: onSelectAll,
                                            isSelected: allRowsSelected,
                                        }}
                                    />
                                    {columns.map((column) => {
                                        return (
                                            <Th key={column.Header} modifier="wrap">
                                                {column.Header}
                                            </Th>
                                        );
                                    })}
                                    <Th aria-label="Row actions" />
                                </Tr>
                            </Thead>
                            <Tbody>
                                {integrations.map((integration, rowIndex) => {
                                    const { id } = integration;
                                    return (
                                        <Tr key={integration.id}>
                                            <Td
                                                key={integration.id}
                                                select={{
                                                    rowIndex,
                                                    onSelect,
                                                    isSelected: selected[rowIndex],
                                                }}
                                            />
                                            {columns.map((column) => {
                                                if (column.Header === 'Name') {
                                                    return (
                                                        <Td key="name">
                                                            <Button
                                                                variant={ButtonVariant.link}
                                                                isInline
                                                                component={LinkShim}
                                                                href={`${integrationsPath}/signatureIntegrations/signature/view/${id}`}
                                                            >
                                                                <TableCellValue
                                                                    row={integration}
                                                                    column={column}
                                                                />
                                                            </Button>
                                                        </Td>
                                                    );
                                                }
                                                return (
                                                    <Td key={column.Header}>
                                                        <TableCellValue
                                                            row={integration}
                                                            column={column}
                                                        />
                                                    </Td>
                                                );
                                            })}
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        </TableComposable>
                    </PageSection>
                </ModalBoxBody>
                <ModalBoxFooter>
                    <Button key="cancel" variant="link" onClick={handleCancelModal}>
                        Cancel
                    </Button>
                    <Button key="confirm" variant="danger" onClick={onConfirmHandler}>
                        Confirm
                    </Button>
                </ModalBoxFooter>
            </Modal>
        </>
    );
}
