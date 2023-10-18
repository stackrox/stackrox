import React from 'react';
import { Modal, ModalBoxBody, pluralize } from '@patternfly/react-core';

import { CveExceptionRequestType } from '../../types';
import DeferralForm, { DeferralFormProps } from './DeferralForm';
import { ScopeContext } from './utils';

export type ExceptionRequestModalOptions = {
    type: CveExceptionRequestType;
    cves: DeferralFormProps['cves'];
} | null;

export type ExceptionRequestModalProps = {
    type: CveExceptionRequestType;
    cves: DeferralFormProps['cves'];
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
        <Modal hasNoBodyWrapper onClose={onClose} title={title} isOpen variant="medium">
            <ModalBoxBody className="pf-u-display-flex pf-u-flex-direction-column">
                {type === 'DEFERRAL' && (
                    <DeferralForm cves={cves} scopeContext={scopeContext} onCancel={onClose} />
                )}
            </ModalBoxBody>
        </Modal>
    );
}

export default ExceptionRequestModal;
