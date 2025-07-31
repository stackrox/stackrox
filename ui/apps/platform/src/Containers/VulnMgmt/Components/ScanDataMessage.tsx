import React, { ReactElement } from 'react';
import { Alert } from '@patternfly/react-core';

import { ScanMessage } from 'messages/vulnMgmt.messages';

// Component props have inconsistent name because ScanMessage is application-specific data structure.
/* eslint-disable generic/react-props-name */
function ScanDataMessage({ header = '', body = '' }: ScanMessage): ReactElement | null {
    return header?.length > 0 || body?.length > 0 ? (
        <div className="px-4 pt-4">
            <Alert variant="warning" isInline title="CVE data may be inaccurate" component="p">
                {header && <p>{header}</p>}
                {body && <p>{body}</p>}
            </Alert>
        </div>
    ) : null;
}
/* eslint-enable-next-line generic/react-props-name */

export default ScanDataMessage;
