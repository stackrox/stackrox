import React from 'react';
import { Alert, AlertActionLink } from '@patternfly/react-core';

export const COMPLIANCE_DISCLAIMER_KEY = 'complianceDisclaimerAccepted';

export type ComplianceUsageDisclaimerProps = {
    className?: string;
    onAccept: () => void;
};

function ComplianceUsageDisclaimer({ className, onAccept }: ComplianceUsageDisclaimerProps) {
    return (
        <>
            <Alert
                className={className}
                variant="info"
                isInline
                title="Usage disclaimer"
                actionLinks={
                    <>
                        <AlertActionLink onClick={onAccept}>
                            I acknowledge that I have read and understand this statement
                        </AlertActionLink>
                    </>
                }
            >
                <p>
                    Red Hat Advanced Cluster Security, and its compliance scanning implementations,
                    assists users by automating the inspection of numerous technical implementations
                    that align with certain aspects of industry standards, benchmarks, and
                    baselines. It does not replace the need for auditors, Qualified Security
                    Assessors, Joint Authorization Boards, or other industry regulatory bodies.
                </p>
            </Alert>
        </>
    );
}

export default ComplianceUsageDisclaimer;
