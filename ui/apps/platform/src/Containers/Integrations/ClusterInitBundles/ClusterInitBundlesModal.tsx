import React, { useState, useRef, MouseEvent, ReactElement } from 'react';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { createStructuredSelector } from 'reselect';
import pluralize from 'pluralize';

import { actions } from 'reducers/clusterInitBundles';
import { selectors } from 'reducers';

import CheckboxTable from 'Components/CheckboxTable';
import { rtTrActionsClassName } from 'Components/Table';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import Modal from 'Components/Modal';
import Dialog from 'Components/Dialog';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import NoResultsMessage from 'Components/NoResultsMessage';
import RowActionButton from 'Components/RowActionButton';
import { ClusterInitBundle } from 'services/ClustersService';

import ClusterInitBundleForm from './ClusterInitBundleForm';
import ClusterInitBundleDetails from './ClusterInitBundleDetails';

export type ClusterInitBundlesModalProps = {
    authProviders: { name: string; id: string }[];
    clusterInitBundles: ClusterInitBundle[];
    clusterInitBundleGenerationWizardOpen: boolean;
    onRequestClose: () => void;
    startClusterInitBundleGenerationWizard: () => void;
    closeClusterInitBundleGenerationWizard: () => void;
    generateClusterInitBundle: () => void;
    revokeClusterInitBundles: (string) => void;
    currentGeneratedClusterInitBundle?: ClusterInitBundle | null;
    currentGeneratedHelmValuesBundle?: ClusterInitBundle | null;
};

function ClusterInitBundlesModal({
    authProviders = [],
    clusterInitBundles,
    clusterInitBundleGenerationWizardOpen,
    onRequestClose,
    startClusterInitBundleGenerationWizard,
    closeClusterInitBundleGenerationWizard,
    generateClusterInitBundle,
    revokeClusterInitBundles,
    currentGeneratedClusterInitBundle = null,
    currentGeneratedHelmValuesBundle = null,
}: ClusterInitBundlesModalProps): ReactElement {
    const [selectedBundleId, setSelectedBundleId] = useState<string | null>(null);
    const [showConfirmationDialog, setShowConfirmationDialog] = useState(false);
    const [selection, setSelection] = useState<string[]>([]);
    const clusterInitBundleModalTable = useRef<CheckboxTable | null>(null);

    function onRowClick(row) {
        setSelectedBundleId(row.id);
    }

    function onSubmit() {
        generateClusterInitBundle();
    }

    function revokeBundles({ id }) {
        if (id) {
            revokeClusterInitBundles([id]);
        } else {
            revokeClusterInitBundles(selection);
            hideConfirmationDialog();
            clearSelection();
        }
    }

    function onRevokeHandler(clusterInitBundle) {
        return (e: MouseEvent<HTMLButtonElement>) => {
            e.stopPropagation();
            revokeBundles(clusterInitBundle);
        };
    }

    function unSelectRow() {
        setSelectedBundleId(null);
    }

    function closeModal() {
        closeClusterInitBundleGenerationWizard();
        onRequestClose();
    }

    function openForm() {
        startClusterInitBundleGenerationWizard();
    }

    function closeForm() {
        closeClusterInitBundleGenerationWizard();
    }

    function clearSelection() {
        setSelection([]);
    }

    function handleShowConfirmationDialog() {
        setShowConfirmationDialog(true);
    }

    function hideConfirmationDialog() {
        setShowConfirmationDialog(false);
    }

    function showModalView() {
        if (!clusterInitBundles || !clusterInitBundles.length) {
            return <NoResultsMessage message="No Cluster Init Bundles Generated" />;
        }

        const columns = [
            { accessor: 'name', Header: 'Name' },
            {
                Header: '',
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: ({ original }) => renderRowActionButtons(original),
            },
        ];

        return (
            <CheckboxTable
                ref={(table) => {
                    clusterInitBundleModalTable.current = table;
                }}
                rows={clusterInitBundles}
                columns={columns}
                onRowClick={onRowClick}
                toggleRow={handleToggleRow}
                toggleSelectAll={handleToggleSelectAll}
                selection={selection}
                selectedRowId={selectedBundleId}
                noDataText="No Cluster Init Bundles Generated"
                minRows={20}
            />
        );
    }

    function handleToggleRow(id) {
        const newSelection = toggleRow(id, selection);
        updateSelection(newSelection);
    }

    function handleToggleSelectAll() {
        const rowsLength = clusterInitBundles.length;
        const tableRef = clusterInitBundleModalTable;
        const newSelection = toggleSelectAll(rowsLength, selection, tableRef?.current?.reactTable);
        updateSelection(newSelection);
    }

    function updateSelection(newSelection) {
        setSelection(newSelection);
    }

    function renderRowActionButtons(clusterInitBundle) {
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100">
                <RowActionButton
                    text="Revoke Cluster Init Bundle"
                    icon={<Icon.Trash2 className="my-1 h-4 w-4" />}
                    onClick={onRevokeHandler(clusterInitBundle)}
                />
            </div>
        );
    }

    function renderPanelButtons() {
        const selectionCount = selection.length;
        return (
            <>
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Slash className="h-4 w-4 ml-1" />}
                        className="btn btn-alert"
                        onClick={handleShowConfirmationDialog}
                        disabled={selectedBundleId !== null}
                        tooltip={`Revoke (${selectionCount})`}
                    >
                        {`Revoke (${selectionCount})`}
                    </PanelButton>
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        className="btn btn-base"
                        onClick={openForm}
                        disabled={
                            clusterInitBundleGenerationWizardOpen || selectedBundleId !== null
                        }
                        tooltip="Generate Cluster Init Bundle"
                    >
                        Generate Bundle
                    </PanelButton>
                )}
            </>
        );
    }

    function renderHeader() {
        return (
            <header className="flex items-center w-full p-4 bg-primary-500 text-base-100 uppercase">
                <span className="flex flex-1">Configure Cluster Init Bundles</span>
                <Icon.X className="h-4 w-4 cursor-pointer" onClick={closeModal} />
            </header>
        );
    }

    function renderTable() {
        const selectionCount = selection.length;
        const clusterInitBundleCount = clusterInitBundles.length;
        const headerText =
            selectionCount !== 0
                ? `${selectionCount} cluster init ${pluralize('bundle', selectionCount)} Selected`
                : `${clusterInitBundleCount} cluster init ${pluralize(
                      'bundle',
                      clusterInitBundleCount
                  )}`;
        return (
            <Panel header={headerText} headerComponents={renderPanelButtons()}>
                {showModalView()}
            </Panel>
        );
    }

    function renderForm() {
        if (!clusterInitBundleGenerationWizardOpen) {
            return null;
        }
        if (currentGeneratedClusterInitBundle) {
            return null;
        }

        const buttons = (
            <PanelButton
                icon={<Icon.Save className="h-4 w-4" />}
                className="btn btn-success mr-2 "
                onClick={onSubmit}
                tooltip="Generate"
            >
                Generate
            </PanelButton>
        );

        return (
            <Panel
                header="Generate Cluster Init Bundle"
                onClose={closeForm}
                headerComponents={buttons}
            >
                <ClusterInitBundleForm />
            </Panel>
        );
    }

    function renderDetails() {
        if (currentGeneratedClusterInitBundle) {
            return (
                <Panel header="Generated Cluster Init Bundle" onClose={closeForm}>
                    <ClusterInitBundleDetails
                        authProviders={authProviders}
                        clusterInitBundle={currentGeneratedClusterInitBundle}
                        helmValuesBundle={currentGeneratedHelmValuesBundle}
                    />
                </Panel>
            );
        }
        if (selectedBundleId) {
            const selectedBundleMetadata = clusterInitBundles.find(
                ({ id }) => id === selectedBundleId
            );
            if (selectedBundleMetadata) {
                return (
                    <Panel header="Cluster Init Bundle Details" onClose={unSelectRow}>
                        <ClusterInitBundleDetails
                            authProviders={authProviders}
                            clusterInitBundle={selectedBundleMetadata}
                            helmValuesBundle={currentGeneratedHelmValuesBundle}
                        />
                    </Panel>
                );
            }
        }
        return null;
    }

    return (
        <Modal isOpen onRequestClose={onRequestClose} className="w-full lg:w-5/6 h-full">
            {renderHeader()}
            <div className="flex flex-1 w-full bg-base-100">
                {renderTable()}
                {renderForm()}
                {renderDetails()}
            </div>
            <Dialog
                isOpen={showConfirmationDialog}
                text={`Are you sure you want to revoke ${selection.length} cluster init ${pluralize(
                    'bundle',
                    selection.length
                )}?`}
                onConfirm={revokeBundles}
                onCancel={hideConfirmationDialog}
            />
        </Modal>
    );
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    clusterInitBundleGenerationWizardOpen: selectors.clusterInitBundleGenerationWizardOpen,
    currentGeneratedClusterInitBundle: selectors.getCurrentGeneratedClusterInitBundle,
    currentGeneratedHelmValuesBundle: selectors.getCurrentGeneratedHelmValuesBundle,
});

const mapDispatchToProps = {
    startClusterInitBundleGenerationWizard: actions.startClusterInitBundleGenerationWizard,
    closeClusterInitBundleGenerationWizard: actions.closeClusterInitBundleGenerationWizard,
    generateClusterInitBundle: actions.generateClusterInitBundle.request as () => void,
    revokeClusterInitBundles: actions.revokeClusterInitBundles,
};

export default connect(mapStateToProps, mapDispatchToProps)(ClusterInitBundlesModal);
