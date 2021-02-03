import React, { useState, useRef, MouseEvent, ReactElement } from 'react';
import { connect } from 'react-redux';
import * as Icon from 'react-feather';
import { createStructuredSelector } from 'reselect';
import pluralize from 'pluralize';
import { ClipLoader } from 'react-spinners';

import { actions } from 'reducers/clusterInitBundles';
import { actions as notificationActions } from 'reducers/notifications';
import { selectors } from 'reducers';

import CheckboxTable from 'Components/CheckboxTable';
import { rtTrActionsClassName } from 'Components/Table';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import Modal from 'Components/Modal';
import CustomDialogue from 'Components/CustomDialogue';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import NoResultsMessage from 'Components/NoResultsMessage';
import RowActionButton from 'Components/RowActionButton';
import { ClusterInitBundle, fetchCAConfig } from 'services/ClustersService';

import FileSaver from 'file-saver';
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
    addToast: (message: string) => void;
    removeToast: () => void;
    currentGeneratedClusterInitBundle?: ClusterInitBundle | null;
    currentGeneratedHelmValuesBundle?: ClusterInitBundle | null;
    currentGeneratedKubectlBundle?: ClusterInitBundle | null;
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
    addToast,
    removeToast,
    currentGeneratedClusterInitBundle = null,
    currentGeneratedHelmValuesBundle = null,
    currentGeneratedKubectlBundle = null,
}: ClusterInitBundlesModalProps): ReactElement {
    const [selectedBundleId, setSelectedBundleId] = useState<string | null>(null);
    const [showConfirmationDialog, setShowConfirmationDialog] = useState(false);
    const [selection, setSelection] = useState<string[]>([]);
    const clusterInitBundleModalTable = useRef<CheckboxTable | null>(null);
    const [downloadingCAConfig, setDownloadingCAConfig] = useState<boolean>(false);

    function onRowClick(row) {
        setSelectedBundleId(row.id);
    }

    function onSubmit() {
        generateClusterInitBundle();
    }

    function onFetchCAConfig() {
        setDownloadingCAConfig(true);
        fetchCAConfig()
            .then((response) => {
                if (!response.helmValuesBundle) {
                    throw Error('server returned no data');
                }
                const bytes = atob(response.helmValuesBundle);
                const file = new Blob([bytes], {
                    type: 'application/x-yaml',
                });
                FileSaver.saveAs(file, 'ca-config.yaml');
            })
            .catch((err: { message: string }) => {
                addToast(`Problem downloading the CA config. Please try again. (${err.message})`);
                setTimeout(removeToast, 5000);
            })
            .finally(() => {
                setDownloadingCAConfig(false);
            });
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
            const newSelection = [clusterInitBundle.id];
            updateSelection(newSelection);
            setShowConfirmationDialog(true);
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
        setSelection([]);
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
                    <>
                        <PanelButton
                            icon={
                                downloadingCAConfig ? (
                                    <ClipLoader loading size={14} />
                                ) : (
                                    <Icon.Save className="h-4 w-4" />
                                )
                            }
                            className="btn-icon btn-tertiary mr-2"
                            onClick={onFetchCAConfig}
                            disabled={downloadingCAConfig}
                            tooltip="Download CA Config (use with pre-created secrets)"
                        >
                            Get CA config
                        </PanelButton>
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
                    </>
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
                        kubectlBundle={currentGeneratedKubectlBundle}
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
                            kubectlBundle={currentGeneratedKubectlBundle}
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
            {showConfirmationDialog && (
                <CustomDialogue
                    className="max-w-3/4 md:max-w-2/3 lg:max-w-1/2"
                    title={`Confirm revoking ${pluralize('bundle', selection.length)}?`}
                    onConfirm={revokeBundles}
                    confirmText="Revoke"
                    onCancel={hideConfirmationDialog}
                    confirmStyle="alert"
                >
                    <div className="overflow-auto p-4">
                        <p className="mb-2">{`Are you sure you want to revoke ${
                            selection.length
                        } cluster init ${pluralize('bundle', selection.length)}?`}</p>
                        <p>
                            <strong>Note:</strong> Revoking a cluster init bundle will cause the
                            StackRox services installed with it in clusters to lose connectivity.
                        </p>
                    </div>
                </CustomDialogue>
            )}
        </Modal>
    );
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    clusterInitBundleGenerationWizardOpen: selectors.clusterInitBundleGenerationWizardOpen,
    currentGeneratedClusterInitBundle: selectors.getCurrentGeneratedClusterInitBundle,
    currentGeneratedHelmValuesBundle: selectors.getCurrentGeneratedHelmValuesBundle,
    currentGeneratedKubectlBundle: selectors.getCurrentGeneratedKubectlBundle,
});

const mapDispatchToProps = {
    startClusterInitBundleGenerationWizard: actions.startClusterInitBundleGenerationWizard,
    closeClusterInitBundleGenerationWizard: actions.closeClusterInitBundleGenerationWizard,
    generateClusterInitBundle: actions.generateClusterInitBundle.request as () => void,
    revokeClusterInitBundles: actions.revokeClusterInitBundles,
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(mapStateToProps, mapDispatchToProps)(ClusterInitBundlesModal);
