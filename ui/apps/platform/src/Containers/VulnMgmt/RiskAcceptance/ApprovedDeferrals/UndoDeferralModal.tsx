/* eslint-disable @typescript-eslint/no-unused-vars */
import { FormResponseMessage } from 'Components/PatternFly/FormMessage';

export type UndoDeferralFormValues = {
    comment: string;
};

export type UndoDeferralModalProps = {
    isOpen: boolean;
    numRequestsToBeAssessed: number;
    onSendRequest: (values: UndoDeferralFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

function UndoDeferralModal({
    isOpen,
    numRequestsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: UndoDeferralModalProps) {
    return null;
}

export default UndoDeferralModal;
