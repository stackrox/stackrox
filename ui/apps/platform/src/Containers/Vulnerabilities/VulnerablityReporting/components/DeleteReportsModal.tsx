import { Alert, AlertGroup } from '@patternfly/react-core';
import pluralize from 'pluralize';

import DeleteModal from 'Components/PatternFly/DeleteModal';

import { isErrorDeleteResult, isSuccessDeleteResult } from '../hooks/useDeleteModal';
import type { DeleteResult } from '../hooks/useDeleteModal';

export type DeleteReportsModalProps = {
    isOpen: boolean;
    onClose: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    reportIdsToDelete: string[];
    deleteResults: DeleteResult[] | null;
    reportConfigurations: { id: string; name: string }[] | null;
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
    const numSuccessfulDeletions = deleteResults?.filter(isSuccessDeleteResult).length || 0;

    return (
        <DeleteModal
            title={`Permanently delete (${reportIdsToDelete.length}) ${pluralize(
                'report',
                reportIdsToDelete.length
            )}?`}
            isOpen={isOpen}
            onClose={onClose}
            isDeleting={isDeleting}
            onDelete={onDelete}
        >
            <AlertGroup>
                {numSuccessfulDeletions > 0 && (
                    <Alert
                        isInline
                        variant="success"
                        title={`Successfully deleted ${numSuccessfulDeletions} ${pluralize(
                            'report',
                            numSuccessfulDeletions
                        )}`}
                        component="p"
                        className="pf-v6-u-mb-sm"
                    />
                )}
                {deleteResults?.filter(isErrorDeleteResult).map((deleteResult) => {
                    const report = reportConfigurations?.find(
                        (reportConfig) => reportConfig.id === deleteResult.id
                    );
                    if (!report) {
                        return null;
                    }
                    return (
                        <Alert
                            isInline
                            variant="danger"
                            title={`Failed to delete "${report.name}"`}
                            component="p"
                            className="pf-v6-u-mb-sm"
                        >
                            {deleteResult.error}
                        </Alert>
                    );
                })}
            </AlertGroup>
            <p>
                The selected report(s) and any attached downloadable reports will be permanently
                deleted. The action cannot be undone.
            </p>
        </DeleteModal>
    );
}

export default DeleteReportsModal;
