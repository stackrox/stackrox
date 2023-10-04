import React from 'react';
import { Modal, pluralize } from '@patternfly/react-core';

import { CveExceptionRequestType } from '../types';

export type ExceptionRequestModalOptions = {
    type: CveExceptionRequestType;
    cves: string[];
} | null;

export type ExceptionRequestModalProps = {
    type: CveExceptionRequestType;
    cves: string[];
    onClose: () => void;
};

function ExceptionRequestModal({ type, cves, onClose }: ExceptionRequestModalProps) {
    const cveCountText = pluralize(cves.length, 'workload CVE');
    const title =
        type === 'DEFERRAL'
            ? `Request deferral for ${cveCountText}`
            : `Mark ${cveCountText} as false positive`;

    return (
        <Modal onClose={onClose} title={title} isOpen variant="medium">
            <div>
                {cves.map((cve) => (
                    <p key={cve}>{cve}</p>
                ))}
            </div>
        </Modal>
    );
}

export default ExceptionRequestModal;
