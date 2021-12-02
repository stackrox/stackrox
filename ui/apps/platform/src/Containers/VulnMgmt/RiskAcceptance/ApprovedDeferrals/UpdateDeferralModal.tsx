/* eslint-disable @typescript-eslint/no-unused-vars */
import { FormResponseMessage } from 'Components/PatternFly/FormMessage';

export type UpdateDeferralFormValues = {
    comment: string;
};

export type UpdateDeferralModalProps = {
    isOpen: boolean;
    numRequestsToBeAssessed: number;
    onSendRequest: (values: UpdateDeferralFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

function UpdateDeferralModal({
    isOpen,
    numRequestsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: UpdateDeferralModalProps) {
    return null;
}

export default UpdateDeferralModal;
