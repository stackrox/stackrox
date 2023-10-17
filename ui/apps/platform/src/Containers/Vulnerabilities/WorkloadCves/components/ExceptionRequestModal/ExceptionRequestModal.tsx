import React from 'react';
import { Modal, pluralize } from '@patternfly/react-core';

import { CveExceptionRequestType } from '../../types';
import DeferralForm from './DeferralForm';
import { ScopeContext } from './utils';

export type ExceptionRequestModalOptions = {
    type: CveExceptionRequestType;
    cves: string[];
} | null;

export type ExceptionRequestModalProps = {
    type: CveExceptionRequestType;
    cves: string[];
    scopeContext: ScopeContext;
    onClose: () => void;
};

function ExceptionRequestModal({ type, cves, scopeContext, onClose }: ExceptionRequestModalProps) {
    const cveCountText = pluralize(cves.length, 'workload CVE');
    const title =
        type === 'DEFERRAL'
            ? `Request deferral for ${cveCountText}`
            : `Mark ${cveCountText} as false positive`;

    return (
        <Modal onClose={onClose} title={title} isOpen variant="medium">
            {type === 'DEFERRAL' && (
                <DeferralForm cves={cves} scopeContext={scopeContext} onCancel={onClose} />
            )}
        </Modal>
    );
}

export default ExceptionRequestModal;
