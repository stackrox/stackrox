import {
    Alert,
    AlertGroup,
    Button,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    Title,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

import type { ReportConfigurationBase } from 'services/ReportsService.types';

import {
    isErrorDeleteResult,
    isSuccessDeleteResult,
} from '../../VulnerablityReporting/hooks/useDeleteModal';
import type { DeleteResult } from '../../VulnerablityReporting/hooks/useDeleteModal';

export type DeleteReportsModalProps = {
    isOpen: boolean;
    onClose: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    reportIdsToDelete: string[];
    deleteResults: DeleteResult[] | null;
    reportConfigurations: ReportConfigurationBase[] | null;
};

function DeleteReportsModal({
    isOpen,
    onClose,
    isDeleting,
    onDelete,
    reportIdsToDelete,
    deleteResults,
    reportConfigurations,
}: DeleteReportsModalProps) {
    const numSuccessfulDeletions = deleteResults?.filter(isSuccessDeleteResult).length ?? 0;
    const title = `Permanently delete ${reportIdsToDelete.length} report ${pluralize(
        'configuration',
        reportIdsToDelete.length
    )}?`;

    return (
        <Modal
            aria-labelledby="delete-reports-modal-title"
            variant="small"
            isOpen={isOpen}
            onClose={onClose}
        >
            <ModalHeader>
                <Title id="delete-reports-modal-title" headingLevel="h2">
                    {title}
                </Title>
            </ModalHeader>
            <ModalBody>
                <AlertGroup>
                    {numSuccessfulDeletions > 0 && (
                        <Alert
                            component="p"
                            isInline
                            title={`Successfully deleted ${numSuccessfulDeletions} report ${pluralize(
                                'configuration',
                                numSuccessfulDeletions
                            )}`}
                            variant="success"
                        />
                    )}
                    {deleteResults?.filter(isErrorDeleteResult).map((deleteResult) => {
                        const reportConfiguration = reportConfigurations?.find(
                            (configuration) => configuration.id === deleteResult.id
                        );
                        if (!reportConfiguration) {
                            return null;
                        }
                        return (
                            <Alert
                                component="p"
                                isInline
                                key={deleteResult.id}
                                title={`Failed to delete "${reportConfiguration.name}"`}
                                variant="danger"
                            >
                                {deleteResult.error}
                            </Alert>
                        );
                    })}
                </AlertGroup>
                <p>
                    The selected report {pluralize('configuration', reportIdsToDelete.length)} and
                    any attached downloadable reports will be permanently deleted. The action cannot
                    be undone.
                </p>
            </ModalBody>
            <ModalFooter>
                <Button
                    variant="danger"
                    isLoading={isDeleting}
                    isDisabled={isDeleting}
                    onClick={onDelete}
                >
                    Delete
                </Button>
                <Button variant="link" onClick={onClose} isDisabled={isDeleting}>
                    Cancel
                </Button>
            </ModalFooter>
        </Modal>
    );
}

export default DeleteReportsModal;
