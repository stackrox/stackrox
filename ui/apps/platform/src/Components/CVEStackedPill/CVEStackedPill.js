import React from 'react';
import PropTypes from 'prop-types';
import { AlertTriangle } from 'react-feather';

import { Tooltip, DetailedTooltipOverlay } from '@stackrox/ui-components';

import FixableCVECount from 'Components/FixableCVECount';
import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';
import getImageScanMessages from 'Containers/VulnMgmt/VulnMgmt.utils/getImageScanMessages';

function PillTooltipBody({ vulnCounter }) {
    if (vulnCounter?.all?.total > 0) {
        const { critical, high, medium, low } = vulnCounter;
        return (
            <div>
                <div>
                    {critical?.total} Critical CVEs ({critical?.fixable} Fixable)
                </div>
                <div>
                    {high?.total} High CVEs ({high?.fixable} Fixable)
                </div>
                <div>
                    {medium?.total} Medium CVEs ({medium?.fixable} Fixable)
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
    imageNotes,
    scan,
}) => {
    const hasCounts = vulnCounter?.all?.total > 0;
    const useScan = !!scan;
    const hasScan = !!scan?.scanTime;

    const pillTooltip = showTooltip
        ? {
              title: 'Criticality Distribution',
              body: <PillTooltipBody vulnCounter={vulnCounter} />,
          }
        : null;

    const width = horizontal ? '' : 'min-w-16';

    const imageScanMessages = getImageScanMessages(imageNotes || [], scan?.notes || []);
    const hasScanMessages = Object.keys(imageScanMessages).length > 0;

    return (
        <div className="flex items-center w-full">
            {useScan && !hasScan && <span>Image not scanned</span>}
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
                        high={vulnCounter.high.total}
                        medium={vulnCounter.medium.total}
                        low={vulnCounter.low.total}
                        tooltip={pillTooltip}
                    />
                </>
            )}
            {hasScanMessages && (
                <Tooltip
                    type="alert"
                    content={
                        <DetailedTooltipOverlay
                            extraClassName="text-alert-800"
                            title="CVE Data May Be Inaccurate"
                            subtitle={imageScanMessages?.header}
                            body={
                                <div className="">
                                    <h3 className="text-font-700">Reason:</h3>
                                    <p className="font-600">{imageScanMessages?.body}</p>
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
        high: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number,
        }),
        medium: PropTypes.shape({
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
    imageNotes: PropTypes.arrayOf(PropTypes.string),
    scan: PropTypes.shape({
        scanTime: PropTypes.string,
        scanNotes: PropTypes.arrayOf(PropTypes.string),
    }),
};

CVEStackedPill.defaultProps = {
    horizontal: false,
    hideLink: false,
    url: '',
    fixableUrl: '',
    showTooltip: true,
    imageNotes: null,
    scan: null,
};

export default CVEStackedPill;
