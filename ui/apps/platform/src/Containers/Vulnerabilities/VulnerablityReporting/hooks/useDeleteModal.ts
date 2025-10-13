import { useState } from 'react';

import useModal from 'hooks/useModal';
import { deleteReportConfiguration } from 'services/ReportsService';
import { Empty } from 'services/types';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type SuccessDeleteResult = {
    success: true;
    id: string;
    result: Empty;
};

type ErrorDeleteResult = {
    success: false;
    id: string;
    error: string;
};

type DeleteResult = SuccessDeleteResult | ErrorDeleteResult;

export type UseDeleteModalProps = {
    onCompleted: () => void;
};

export type UseDeleteModalResult = {
    openDeleteModal: (reportIds: string[]) => void;
    isDeleteModalOpen: boolean;
    closeDeleteModal: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    deleteResults: DeleteResult[] | null;
    reportIdsToDelete: string[];
};

export function isSuccessDeleteResult(
    deleteResult: DeleteResult
): deleteResult is SuccessDeleteResult {
    return deleteResult.success === true;
}

export function isErrorDeleteResult(deleteResult: DeleteResult): deleteResult is ErrorDeleteResult {
    return deleteResult.success === false;
}

function useDeleteModal({ onCompleted }: UseDeleteModalProps): UseDeleteModalResult {
    const { isModalOpen: isDeleteModalOpen, openModal, closeModal } = useModal();
    const [reportIdsToDelete, setReportIdsToDelete] = useState<string[]>([]);
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteResults, setDeleteResults] = useState<DeleteResult[] | null>(null);

    function openDeleteModal(reportIds: string[]) {
        openModal();
        setReportIdsToDelete(reportIds);
    }

    function closeDeleteModal() {
        closeModal();
        setReportIdsToDelete([]);
        setIsDeleting(false);
        setDeleteResults(null);
    }

    // This function removes a report configuration and then returns
    // a result object that captures the outcome, including any
    // potential errors.
    async function onSafeDelete(id: string): Promise<DeleteResult> {
        try {
            const result = await deleteReportConfiguration(id);
            return { success: true, id, result };
        } catch (error) {
            return {
                success: false,
                id,
                error: getAxiosErrorMessage(error),
            };
        }
    }

    async function onDelete() {
        setIsDeleting(true);
        const results = await Promise.all(reportIdsToDelete.map((id) => onSafeDelete(id)));
        const hasSuccessfulDeletes = results.some((result) => result.success);
        const hasErrors = results.some((result) => !result.success);
        setIsDeleting(false);
        if (!hasErrors) {
            closeDeleteModal();
        } else {
            // Continue monitoring the report configurations that failed to delete.
            const newReportIdsToDelete = results
                .filter((result) => !result.success)
                .map((result) => result.id);
            setReportIdsToDelete(newReportIdsToDelete);
            setDeleteResults(results);
        }
        if (hasSuccessfulDeletes) {
            // We need to ensure the list is refreshed to reflect the report configurations that were successfully deleted.
            onCompleted();
        }
    }

    return {
        openDeleteModal,
        isDeleteModalOpen,
        closeDeleteModal,
        isDeleting,
        onDelete,
        deleteResults,
        reportIdsToDelete,
    };
}

export default useDeleteModal;
