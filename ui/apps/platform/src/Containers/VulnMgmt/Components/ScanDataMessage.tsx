import React, { ReactElement } from 'react';
import { Alert } from '@patternfly/react-core';

import { ScanMessage } from 'messages/vulnMgmt.messages';

function ScanDataMessage({ header = '', body = '' }: ScanMessage): ReactElement | null {
    return header?.length > 0 || body?.length > 0 ? (
        <div className="px-4 pt-4">
            <Alert variant="warning" isInline title="CVE data may be inaccurate" component="h3">
                {header && <p>{header}</p>}
                {body && <p>{body}</p>}
            </Alert>
        </div>
    ) : null;
}

export default ScanDataMessage;
