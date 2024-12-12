import { useState } from 'react';

import useModal from 'hooks/useModal';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { Empty } from 'services/types';

export type UseDeleteDownloadModalProps = {
    deleteDownloadFunc: (reportId: string) => Promise<Empty>;
    onCompleted: () => void;
};

export type UseDeleteDownloadModalResult = {
    openDeleteDownloadModal: (reportJobId: string) => void;
    isDeleteDownloadModalOpen: boolean;
    closeDeleteDownloadModal: () => void;
    isDeletingDownload: boolean;
    onDeleteDownload: () => void;
    deleteDownloadError: string | null;
};

function useDeleteDownloadModal({
    deleteDownloadFunc,
    onCompleted,
}: UseDeleteDownloadModalProps): UseDeleteDownloadModalResult {
    const { isModalOpen, openModal, closeModal } = useModal();
    const [reportJobIdToDeleteDownload, setReportIdToDeleteDownload] = useState<string>('');
    const [isDeletingDownload, setIsDeletingDownload] = useState(false);
    const [deleteDownloadError, setDeleteDownloadError] = useState<string | null>(null);

    function openDeleteDownloadModal(reportJobId: string) {
        openModal();
        setReportIdToDeleteDownload(reportJobId);
    }

    function closeDeleteDownloadModal() {
        closeModal();
        setReportIdToDeleteDownload('');
        setIsDeletingDownload(false);
        setDeleteDownloadError(null);
    }

    function onDeleteDownload() {
        setIsDeletingDownload(true);
        deleteDownloadFunc(reportJobIdToDeleteDownload)
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
