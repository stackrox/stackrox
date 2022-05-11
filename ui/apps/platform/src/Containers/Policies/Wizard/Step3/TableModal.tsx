import React, { useState } from 'react';
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
import isEqual from 'lodash/isEqual';
import pluralize from 'pluralize';

import LinkShim from 'Components/PatternFly/LinkShim';
import useTableSelection from 'hooks/useTableSelection';
import TableCellValue from 'Components/TableCellValue/TableCellValue';

type TableModalProps = {
    setValue: (value: any) => void;
    value: any;
    readOnly?: boolean;
    rows: any;
    columns: any;
    typeText: string;
};

function TableModal({
    setValue,
    value,
    readOnly = false,
    rows,
    columns,
    typeText,
}: TableModalProps): React.ReactElement {
    const [isModalOpen, setIsModalOpen] = useState(false);

    const { selected, onSelect, onSelectAll, allRowsSelected, onResetAll, getSelectedIds } =
        useTableSelection(rows, (row) => {
            return value.arrayValue ? (value.arrayValue.includes(row.id) as boolean) : false;
        });

    function onCloseModalHandler() {
        onResetAll();
        setIsModalOpen(false);
    }

    function onSaveHandler() {
        setValue({ arrayValue: getSelectedIds() });
        setIsModalOpen(false);
    }

    return (
        <>
            <TextInput
                id="table-modal-text-input"
                isDisabled
                value={
                    value.arrayValue?.length > 0
                        ? `Selected ${value.arrayValue?.length as string} ${pluralize(
                              typeText,
                              value.arrayValue?.length
                          )}`
                        : `Add ${typeText}s`
                }
            />
            <Button
                key="open-select-modal"
                variant={ButtonVariant.primary}
                onClick={() => {
                    setIsModalOpen(true);
                }}
            >
                {readOnly ? 'View' : 'Select'}
            </Button>
            <Modal
                title={`Add ${typeText}s to policy criteria`}
                isOpen={isModalOpen}
                variant={ModalVariant.large}
                onClose={onCloseModalHandler}
                data-testid="select-table-modal"
                aria-label={`Select ${typeText}s modal`}
                hasNoBodyWrapper
            >
                <ModalBoxBody>
                    <PageSection variant="light">
                        {!!rows.length && (
                            <>
                                Select {typeText}s from the table below.
                                <TableComposable variant="compact" isStickyHeader>
                                    <Thead>
                                        <Tr>
                                            <Th
                                                select={{
                                                    onSelect: onSelectAll,
                                                    isSelected: allRowsSelected,
                                                    isHeaderSelectDisabled: readOnly,
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
                                        {rows.map((row, rowIndex) => {
                                            const { id, link } = row;
                                            return (
                                                <Tr key={id}>
                                                    <Td
                                                        key={id}
                                                        select={{
                                                            rowIndex,
                                                            onSelect,
                                                            isSelected: selected[rowIndex],
                                                            disable: readOnly,
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
                                                                        href={link}
                                                                    >
                                                                        <TableCellValue
                                                                            row={row}
                                                                            column={column}
                                                                        />
                                                                    </Button>
                                                                </Td>
                                                            );
                                                        }
                                                        return (
                                                            <Td key={column.Header}>
                                                                <TableCellValue
                                                                    row={row}
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
                            </>
                        )}
                        {!rows.length && (
                            <div>Please configure {typeText}s to add them as policy criteria.</div>
                        )}
                    </PageSection>
                </ModalBoxBody>
                <ModalBoxFooter>
                    <Button
                        key="save"
                        variant="primary"
                        onClick={onSaveHandler}
                        isDisabled={
                            readOnly || isEqual(value.arrayValue, getSelectedIds()) || !rows.length
                        }
                    >
                        Save
                    </Button>
                    <Button key="cancel" variant="secondary" onClick={onCloseModalHandler}>
                        Cancel
                    </Button>
                </ModalBoxFooter>
            </Modal>
        </>
    );
}

export default TableModal;
