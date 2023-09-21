import { Dispatch, SetStateAction, useState } from 'react';
import { DeliveryDestination } from '../forms/useReportFormValues';

export type UseEmailTemplateModalResult = {
    isEmailTemplateModalOpen: boolean;
    closeEmailTemplateModal: () => void;
    selectedEmailSubject: string;
    selectedEmailBody: string;
    selectedDeliveryDestination: DeliveryDestination | null;
    setSelectedDeliveryDestination: Dispatch<SetStateAction<DeliveryDestination | null>>;
};

function useEmailTemplateModal(): UseEmailTemplateModalResult {
    const [selectedDeliveryDestination, setSelectedDeliveryDestination] =
        useState<DeliveryDestination | null>(null);

    function closeEmailTemplateModal() {
        setSelectedDeliveryDestination(null);
    }

    return {
        isEmailTemplateModalOpen: !!selectedDeliveryDestination,
        closeEmailTemplateModal,
        selectedEmailSubject: selectedDeliveryDestination?.customSubject || '',
        selectedEmailBody: selectedDeliveryDestination?.customBody || '',
        selectedDeliveryDestination,
        setSelectedDeliveryDestination,
    };
}

export default useEmailTemplateModal;
