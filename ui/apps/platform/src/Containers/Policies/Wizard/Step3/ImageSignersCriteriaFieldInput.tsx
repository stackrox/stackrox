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
import isEqual from 'lodash/isEqual';
import LinkShim from 'Components/PatternFly/LinkShim';
import { integrationsPath } from 'routePaths';
import { fetchSignatureIntegrations } from 'services/SignatureIntegrationsService';
import { SignatureIntegration } from 'types/signatureIntegration.proto';
import useTableSelection from 'hooks/useTableSelection';
import TableCellValue from 'Components/TableCellValue/TableCellValue';
import tableColumnDescriptor from 'Containers/Integrations/utils/tableColumnDescriptor';

function ImageSignersCriteriaFieldInput({ setValue, value, readOnly = false }): React.ReactElement {
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [integrations, setIntegrations] = useState<SignatureIntegration[]>([]);
    const columns = [...tableColumnDescriptor.signatureIntegrations.signature];

    useEffect(() => {
        fetchSignatureIntegrations()
            .then((data) => {
                setIntegrations(data);
            })
            .catch(() => {
                setIntegrations([]);
            });
    }, []);

    const { selected, onSelect, onSelectAll, allRowsSelected, onResetAll, getSelectedIds } =
        useTableSelection<SignatureIntegration>(integrations, (integration) => {
            return value.arrayValue
                ? (value.arrayValue.includes(integration.id) as boolean)
                : false;
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
                id="image-signers-text-input"
                isDisabled
                value={
                    value.arrayValue?.length > 0
                        ? `Selected ${value.arrayValue?.length as number} trusted image signers`
                        : 'Add trusted image signers'
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
                title="Add trusted image signers to policy criteria"
                isOpen={isModalOpen}
                variant={ModalVariant.large}
                onClose={onCloseModalHandler}
                data-testid="select-image-signers-modal"
                aria-label="Select image signers modal"
                hasNoBodyWrapper
            >
                <ModalBoxBody>
                    <PageSection variant="light">
                        Select trusted image signers from the table below.
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
                    <Button
                        key="save"
                        variant="primary"
                        onClick={onSaveHandler}
                        isDisabled={readOnly || isEqual(value.arrayValue, getSelectedIds())}
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

export default ImageSignersCriteriaFieldInput;
