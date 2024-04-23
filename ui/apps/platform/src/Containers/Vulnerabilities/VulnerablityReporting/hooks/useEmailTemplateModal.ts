import { Dispatch, SetStateAction, useState } from 'react';
import { NotifierConfiguration } from 'services/ReportsService.types';

export type UseEmailTemplateModalResult = {
    isEmailTemplateModalOpen: boolean;
    closeEmailTemplateModal: () => void;
    selectedEmailSubject: string;
    selectedEmailBody: string;
    selectedDeliveryDestination: NotifierConfiguration | null;
    setSelectedDeliveryDestination: Dispatch<SetStateAction<NotifierConfiguration | null>>;
};

function useEmailTemplateModal(): UseEmailTemplateModalResult {
    const [selectedDeliveryDestination, setSelectedDeliveryDestination] =
        useState<NotifierConfiguration | null>(null);

    function closeEmailTemplateModal() {
        setSelectedDeliveryDestination(null);
    }

    return {
        isEmailTemplateModalOpen: !!selectedDeliveryDestination,
        closeEmailTemplateModal,
        selectedEmailSubject: selectedDeliveryDestination?.emailConfig?.customSubject ?? '',
        selectedEmailBody: selectedDeliveryDestination?.emailConfig?.customBody ?? '',
        selectedDeliveryDestination,
        setSelectedDeliveryDestination,
    };
}

export default useEmailTemplateModal;
