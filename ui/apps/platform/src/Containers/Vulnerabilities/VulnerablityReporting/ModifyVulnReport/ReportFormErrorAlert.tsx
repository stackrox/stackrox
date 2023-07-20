import React, { useEffect } from 'react';
import { Alert, AlertVariant } from '@patternfly/react-core';

function ReportFormErrorAlert({ error }) {
    const alertRef = React.useRef<HTMLInputElement>(null);

    // When an error occurs, scroll the message into view
    useEffect(() => {
        if (error && alertRef.current) {
            alertRef.current?.scrollIntoView({
                behavior: 'smooth',
            });
        }
    }, [error]);

    return (
        <div ref={alertRef}>
            {error && (
                <Alert
                    isInline
                    variant={AlertVariant.danger}
                    title={error}
                    className="pf-u-mb-sm"
                />
            )}
        </div>
    );
}

export default ReportFormErrorAlert;
