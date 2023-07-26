import useModal from 'hooks/useModal';
import { useState } from 'react';
import { deleteReportConfiguration } from 'services/ReportsService';
import { getErrorMessage } from '../errorUtils';

export type UseDeleteModalProps = {
    onCompleted: () => void;
};

export type UseDeleteModalResult = {
    openDeleteModal: (reportId: string) => void;
    isDeleteModalOpen: boolean;
    closeDeleteModal: () => void;
    isDeleting: boolean;
    onDelete: () => void;
    deleteError: string | null;
};

function useDeleteModal({ onCompleted }: UseDeleteModalProps): UseDeleteModalResult {
    const { isModalOpen: isDeleteModalOpen, openModal, closeModal } = useModal();
    const [reportIdToDelete, setReportIdToDelete] = useState<string>('');
    const [isDeleting, setIsDeleting] = useState(false);
    const [deleteError, setDeleteError] = useState<string | null>(null);

    function openDeleteModal(reportId: string) {
        openModal();
        setReportIdToDelete(reportId);
    }

    function closeDeleteModal() {
        closeModal();
        setReportIdToDelete('');
        setIsDeleting(false);
        setDeleteError(null);
    }

    async function onDelete() {
        setIsDeleting(true);
        try {
            await deleteReportConfiguration(reportIdToDelete);
            setIsDeleting(false);
            closeDeleteModal();
            onCompleted();
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
    };
}

export default useDeleteModal;
