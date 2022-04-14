import React from 'react';
import PropTypes from 'prop-types';
import { AlertTriangle } from 'react-feather';

import { Tooltip, DetailedTooltipOverlay } from '@stackrox/ui-components';

import FixableCVECount from 'Components/FixableCVECount';
import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';

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
    scan,
    scanMessage,
}) => {
    const hasCounts = vulnCounter?.all?.total > 0;
    const useScan = !!scan;
    const hasScan = !!scan?.scanTime;
    const hasScanMessage = !!scanMessage?.header;

    const pillTooltip = showTooltip
        ? {
              title: 'Criticality Distribution',
              body: <PillTooltipBody vulnCounter={vulnCounter} />,
          }
        : null;

    const width = horizontal ? '' : 'min-w-16';

    return (
        <div className="flex items-center w-full">
            {useScan && !hasScan && <span>{entityName} not scanned</span>}
            {!hasCounts && <span>No CVEs</span>}
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
                    <SeverityStackedPill
                        critical={vulnCounter.critical.total}
                        important={vulnCounter.important.total}
                        moderate={vulnCounter.moderate.total}
                        low={vulnCounter.low.total}
                        tooltip={pillTooltip}
                    />
                </>
            )}
            {hasScanMessage && (
                <Tooltip
                    type="alert"
                    content={
                        <DetailedTooltipOverlay
                            extraClassName="text-alert-800"
                            title="CVE Data May Be Inaccurate"
                            subtitle={scanMessage?.header}
                            body={
                                <div className="">
                                    <h3 className="text-font-700">Reason:</h3>
                                    <p className="font-600">{scanMessage?.body}</p>
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
    scan: PropTypes.shape({
        scanTime: PropTypes.string,
        notes: PropTypes.arrayOf(PropTypes.string),
    }),
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
    scan: null,
    scanMessage: null,
};

export default CVEStackedPill;
