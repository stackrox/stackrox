import React from 'react';
import PropTypes from 'prop-types';
import { AlertTriangle } from 'react-feather';
import { Tooltip } from '@patternfly/react-core';

import DetailedTooltipContent from 'Components/DetailedTooltipContent';
import FixableCVECount from 'Components/FixableCVECount';

import SeverityStackedPill from './SeverityStackedPill';

function PillTooltipBody({ vulnCounter }) {
    if (vulnCounter?.all?.total > 0) {
        const { critical, important, moderate, low } = vulnCounter;
        return (
            <div>
                <div>
                    {critical?.total} Critical CVEs ({critical?.fixable} Fixable)
                </div>
                <div>
                    {important?.total} Important CVEs ({important?.fixable} Fixable)
                </div>
                <div>
                    {moderate?.total} Moderate CVEs ({moderate?.fixable} Fixable)
                </div>
                <div>
                    {low?.total} Low CVEs ({low?.fixable} Fixable)
                </div>
            </div>
        );
    }

    return null;
}
const CVEStackedPill = ({
    horizontal,
    vulnCounter,
    hideLink,
    url,
    fixableUrl,
    showTooltip,
    entityName,
    scanTime,
    scanMessage,
}) => {
    const hasCounts = vulnCounter?.all?.total > 0;
    const useScan = scanTime !== '-';
    const hasScan = !!scanTime;
    const hasScanMessage = !!scanMessage?.header;

    const width = horizontal ? '' : 'min-w-16';

    return (
        <div className="flex items-center w-full">
            {useScan && !hasScan && <span>{entityName} not scanned</span>}
            {(!useScan || hasScan) && !hasCounts && <span>No CVEs</span>}
            {hasCounts && (
                <>
                    <div className={`mr-2 ${width}`}>
                        <FixableCVECount
                            cves={vulnCounter.all.total}
                            fixable={vulnCounter.all.fixable}
                            orientation={horizontal ? 'horizontal' : 'vertical'}
                            url={url}
                            fixableUrl={fixableUrl}
                            hideLink={hideLink}
                        />
                    </div>
                    {showTooltip ? (
                        <Tooltip
                            isContentLeftAligned
                            content={
                                <DetailedTooltipContent
                                    title="Severity distribution"
                                    body={<PillTooltipBody vulnCounter={vulnCounter} />}
                                />
                            }
                        >
                            <SeverityStackedPill vulnCounter={vulnCounter} />
                        </Tooltip>
                    ) : (
                        <SeverityStackedPill vulnCounter={vulnCounter} />
                    )}
                </>
            )}
            {hasScanMessage && (
                <Tooltip
                    isContentLeftAligned
                    content={
                        <DetailedTooltipContent
                            title="CVE Data May Be Inaccurate"
                            subtitle={scanMessage?.header}
                            body={
                                <div className="">
                                    <h3 className="font-700">Reason:</h3>
                                    <p>{scanMessage?.body}</p>
                                </div>
                            }
                        />
                    }
                >
                    <AlertTriangle className="w-4 h-4 text-alert-700 ml-2" />
                </Tooltip>
            )}
        </div>
    );
};

CVEStackedPill.propTypes = {
    horizontal: PropTypes.bool,
    vulnCounter: PropTypes.shape({
        critical: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number,
        }),
        important: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number,
        }),
        moderate: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number,
        }),
        low: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number,
        }),
        all: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number,
        }),
    }).isRequired,
    hideLink: PropTypes.bool,
    url: PropTypes.string,
    fixableUrl: PropTypes.string,
    showTooltip: PropTypes.bool,
    entityName: PropTypes.string,
    scanTime: PropTypes.string,
    scanMessage: PropTypes.shape({
        header: PropTypes.string,
        body: PropTypes.string,
    }),
};

CVEStackedPill.defaultProps = {
    horizontal: false,
    hideLink: false,
    url: '',
    fixableUrl: '',
    showTooltip: true,
    entityName: '',
    scanTime: '-',
    scanMessage: null,
};

export default CVEStackedPill;
