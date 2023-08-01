import useModal from 'hooks/useModal';
import { useState } from 'react';
import { deleteDownloadableReport } from 'services/ReportsService';
import { getErrorMessage } from '../errorUtils';

export type UseDeleteDownloadModalProps = {
    onCompleted: () => void;
};

export type UseDeleteDownloadModalResult = {
    openDeleteDownloadModal: (reportId: string) => void;
    isDeleteDownloadModalOpen: boolean;
    closeDeleteDownloadModal: () => void;
    isDeletingDownload: boolean;
    onDeleteDownload: () => void;
    deleteDownloadError: string | null;
};

function useDeleteDownloadModal({
    onCompleted,
}: UseDeleteDownloadModalProps): UseDeleteDownloadModalResult {
    const { isModalOpen, openModal, closeModal } = useModal();
    const [reportIdToDeleteDownload, setReportIdToDeleteDownload] = useState<string>('');
    const [isDeletingDownload, setIsDeletingDownload] = useState(false);
    const [deleteDownloadError, setDeleteDownloadError] = useState<string | null>(null);

    function openDeleteDownloadModal(reportId: string) {
        openModal();
        setReportIdToDeleteDownload(reportId);
    }

    function closeDeleteDownloadModal() {
        closeModal();
        setReportIdToDeleteDownload('');
        setIsDeletingDownload(false);
        setDeleteDownloadError(null);
    }

    async function onDeleteDownload() {
        setIsDeletingDownload(true);
        try {
            await deleteDownloadableReport(reportIdToDeleteDownload);
            setIsDeletingDownload(false);
            closeDeleteDownloadModal();
            onCompleted();
        } catch (err) {
            setIsDeletingDownload(false);
            setDeleteDownloadError(getErrorMessage(err));
        }
    }

    return {
        openDeleteDownloadModal,
        isDeleteDownloadModalOpen: isModalOpen,
        closeDeleteDownloadModal,
        isDeletingDownload,
        onDeleteDownload,
        deleteDownloadError,
    };
}

export default useDeleteDownloadModal;
