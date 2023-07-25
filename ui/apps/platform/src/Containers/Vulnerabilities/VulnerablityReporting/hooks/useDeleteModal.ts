import useModal from 'hooks/useModal';
import { useState } from 'react';
import { deleteReportConfiguration } from 'services/ReportsService';
import { getErrorMessage } from '../errorUtils';

export type UseDeleteModalResult = {
    openDeleteModal: (reportId: string) => void;
    isDeleteModalOpen: boolean;
    closeDeleteModal: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    deleteError: string | null;
    isDeleted: boolean;
};

function useDeleteModal(): UseDeleteModalResult {
    const { isModalOpen: isDeleteModalOpen, openModal, closeModal } = useModal();
    const [reportIdToDelete, setReportIdToDelete] = useState<string>('');
    const [isDeleting, setIsDeleting] = useState(false);
    const [isDeleted, setIsDeleted] = useState(false);
    const [deleteError, setDeleteError] = useState<string | null>(null);

    function openDeleteModal(reportId: string) {
        openModal();
        setReportIdToDelete(reportId);
    }

    function closeDeleteModal() {
        closeModal();
        setReportIdToDelete('');
    }

    async function onDelete() {
        setIsDeleting(true);
        try {
            await deleteReportConfiguration(reportIdToDelete);
            setIsDeleting(false);
            setIsDeleted(true);
            closeDeleteModal();
        } catch (err) {
            setIsDeleting(false);
            setDeleteError(getErrorMessage(err));
        }
    }

    return {
        openDeleteModal,
        isDeleteModalOpen,
        closeDeleteModal,
        isDeleting,
        onDelete,
        deleteError,
        isDeleted,
    };
}

export default useDeleteModal;
