import { useState } from 'react';
import { BaseVulnerabilityException } from 'services/VulnerabilityExceptionService';
import { ExceptionRequestModalOptions } from '../components/ExceptionRequestModal/ExceptionRequestModal';

export type ShowExceptionModalAction =
    | {
          type: 'DEFERRAL' | 'FALSE_POSITIVE';
          cves: { cve: string; summary: string }[];
      }
    | {
          type: 'COMPLETION';
          exception: BaseVulnerabilityException;
      };

export type UseExceptionRequestModalReturn = {
    exceptionRequestModalOptions: ExceptionRequestModalOptions | null;
    completedException: BaseVulnerabilityException | null;
    showModal: (action: ShowExceptionModalAction) => void;
    closeModals: () => void;
    createExceptionModalActions: (options: {
        cve: string;
        summary: string;
    }) => { title: string; onClick: () => void }[];
};

/**
 * Manages the state of the exception request modal and the completion modal.
 */
export default function useExceptionRequestModal(): UseExceptionRequestModalReturn {
    const [exceptionRequestOptions, setExceptionRequestOptions] =
        useState<ExceptionRequestModalOptions | null>(null);

    const [completedException, setCompletedException] = useState<BaseVulnerabilityException | null>(
        null
    );

    function showModal(action: ShowExceptionModalAction) {
        if (action.type === 'COMPLETION') {
            setExceptionRequestOptions(null);
            setCompletedException(action.exception);
        } else {
            setCompletedException(null);
            setExceptionRequestOptions(action);
        }
    }

    return {
        exceptionRequestModalOptions: exceptionRequestOptions,
        completedException,
        showModal,
        closeModals: () => {
            setExceptionRequestOptions(null);
            setCompletedException(null);
        },
        createExceptionModalActions: ({ cve, summary }) => [
            {
                title: 'Defer CVE',
                onClick: () => showModal({ type: 'DEFERRAL', cves: [{ cve, summary }] }),
            },
            {
                title: 'Mark as false positive',
                onClick: () => showModal({ type: 'FALSE_POSITIVE', cves: [{ cve, summary }] }),
            },
        ],
    };
}
