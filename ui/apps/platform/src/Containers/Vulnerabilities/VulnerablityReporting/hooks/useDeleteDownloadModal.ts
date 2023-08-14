import { useState } from 'react';

import useModal from 'hooks/useModal';
import { deleteDownloadableReport } from 'services/ReportsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

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

    function onDeleteDownload() {
        setIsDeletingDownload(true);
        deleteDownloadableReport(reportIdToDeleteDownload)
            .then(() => {
                closeDeleteDownloadModal();
                onCompleted();
            })
            .catch((error) => {
                const message = getAxiosErrorMessage(error);
                setDeleteDownloadError(message);
            })
            .finally(() => {
                setIsDeletingDownload(false);
            });
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
