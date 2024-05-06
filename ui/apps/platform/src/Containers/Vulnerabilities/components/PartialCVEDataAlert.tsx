import React from 'react';
import { Alert } from '@patternfly/react-core';

function PartialCVEDataAlert() {
    return <Alert isInline component="p" variant="warning" title="Partial CVE data" />;
}

export default PartialCVEDataAlert;
