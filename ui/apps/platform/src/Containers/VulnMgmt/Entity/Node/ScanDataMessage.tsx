import React, { ReactElement } from 'react';
import { Message } from '@stackrox/ui-components';

import getNodeScanMessages from 'Containers/VulnMgmt/VulnMgmt.utils/getNodeScanMessages';

function ScanDataMessage({ nodeNotes = [], scanNotes = [] }): ReactElement | null {
    const nodeScanMessages = getNodeScanMessages(nodeNotes || [], scanNotes || []);

    return Object.keys(nodeScanMessages).length > 0 ? (
        <div className="px-4 pt-4">
            <Message type="error">
                <div className="w-full">
                    <header className="text-lg pb-2 border-b border-alert-300 mb-2 w-full">
                        <h2 className="mb-1 font-700 tracking-wide uppercase">
                            CVE Data May Be Inaccurate
                        </h2>
                        <span>{nodeScanMessages.header}</span>
                    </header>
                    <p>{nodeScanMessages.body}</p>
                </div>
            </Message>
        </div>
    ) : null;
}

export default ScanDataMessage;
