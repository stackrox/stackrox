import React, { ReactElement } from 'react';
import { Message } from '@stackrox/ui-components';

import { ScanMessage } from 'messages/vulnMgmt.messages';

function ScanDataMessage({ header = '', body = '' }: ScanMessage): ReactElement | null {
    return header?.length > 0 || body?.length > 0 ? (
        <div className="px-4 pt-4">
            <Message type="error">
                <div className="w-full">
                    <header className="text-lg pb-2 border-b border-alert-300 mb-2 w-full">
                        <h2 className="mb-1 font-700 tracking-wide uppercase">
                            CVE Data May Be Inaccurate
                        </h2>
                        <span>{header}</span>
                    </header>
                    <p>{body}</p>
                </div>
            </Message>
        </div>
    ) : null;
}

export default ScanDataMessage;
