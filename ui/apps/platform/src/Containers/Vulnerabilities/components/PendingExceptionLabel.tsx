import React from 'react';

import { Link } from 'react-router-dom-v5-compat';
import { Label, Popover } from '@patternfly/react-core';
import { OutlinedClockIcon } from '@patternfly/react-icons';

import { getProductBranding } from 'constants/productBranding';
import PopoverBodyContent from 'Components/PopoverBodyContent';
import type { VulnerabilityState } from 'types/cve.proto';
import useWorkloadCveViewContext from '../WorkloadCves/hooks/useWorkloadCveViewContext';

export type PendingExceptionLabelProps = {
    cve: string;
    isCompact: boolean; // true for table
    vulnerabilityState: VulnerabilityState;
};

/**
 * 'Pending exception' label layout for use in tables. Conditionally renders a label
 * with a link to the exception request page if the vulnerability has a pending exception.
 * In the ConsolePlugin context (when exceptionDetails URL is not available), renders a
 * popover instead with information about vulnerability exceptions.
 *
 * @param cve - The CVE ID of the vulnerability
 * @param isCompact - Whether to render the label in compact mode (true for tables)
 * @param vulnerabilityState - The vulnerability state
 */
function PendingExceptionLabel({ cve, isCompact, vulnerabilityState }: PendingExceptionLabelProps) {
    const { urlBuilder } = useWorkloadCveViewContext();
    const labelText = vulnerabilityState === 'OBSERVED' ? 'Pending exception' : 'Pending update';

    const { shortName } = getProductBranding();

    const label = (
        <Label
            color="blue"
            isCompact={isCompact}
            icon={<OutlinedClockIcon />}
            variant="outline"
            style={!urlBuilder.exceptionDetails ? { cursor: 'pointer' } : undefined}
        >
            {urlBuilder.exceptionDetails ? (
                <Link to={urlBuilder.exceptionDetails(cve)}>{labelText}</Link>
            ) : (
                labelText
            )}
        </Label>
    );

    // In ConsolePlugin context, wrap with a popover since there's no detail page
    if (!urlBuilder.exceptionDetails) {
        return (
            <Popover
                aria-label="Pending exception information"
                bodyContent={
                    <PopoverBodyContent
                        headerContent={labelText}
                        bodyContent={`This vulnerability has a pending exception request. Exception management is available in the standalone ${shortName} interface.`}
                    />
                }
                enableFlip
                position="top"
            >
                {label}
            </Popover>
        );
    }

    return label;
}

export default PendingExceptionLabel;
