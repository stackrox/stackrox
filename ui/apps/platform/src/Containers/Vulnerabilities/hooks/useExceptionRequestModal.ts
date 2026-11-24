import { useState } from 'react';
import type { BaseVulnerabilityException } from 'services/VulnerabilityExceptionService';
import type { ExceptionRequestModalOptions } from '../components/ExceptionRequestModal/ExceptionRequestModal';

export type ShowExceptionModalAction =
    | {
          type: 'DEFERRAL' | 'FALSE_POSITIVE';
          cves: { cve: string; summary: string; numAffectedImages: number }[];
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
        numAffectedImages: number;
    }) => { title: string; onClick: () => void }[];
};

export type UseExceptionRequestModalOptions = {
    // Optional loader for the CVE summary when the caller only has an empty placeholder.
    resolveSummary?: (cve: string) => Promise<string>;
};

/**
 * Manages the state of the exception request modal and the completion modal.
 */
export default function useExceptionRequestModal(
    options: UseExceptionRequestModalOptions = {}
): UseExceptionRequestModalReturn {
    const [exceptionRequestOptions, setExceptionRequestOptions] =
        useState<ExceptionRequestModalOptions | null>(null);

    const [completedException, setCompletedException] = useState<BaseVulnerabilityException | null>(
        null
    );

    async function withSummary(cve: string, summary: string, numAffectedImages: number) {
        if (summary || !options.resolveSummary) {
            return { cve, summary, numAffectedImages };
        }
        const resolvedSummary = await options.resolveSummary(cve).catch(() => '');
        return { cve, summary: resolvedSummary, numAffectedImages };
    }

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
        createExceptionModalActions: ({ cve, summary, numAffectedImages }) => [
            {
                title: 'Defer CVE',
                onClick: async () =>
                    showModal({
                        type: 'DEFERRAL',
                        cves: [await withSummary(cve, summary, numAffectedImages)],
                    }),
            },
            {
                title: 'Mark as false positive',
                onClick: async () =>
                    showModal({
                        type: 'FALSE_POSITIVE',
                        cves: [await withSummary(cve, summary, numAffectedImages)],
                    }),
            },
        ],
    };
}
