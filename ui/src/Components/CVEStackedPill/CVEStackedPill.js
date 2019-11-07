import React from 'react';
import PropTypes from 'prop-types';
import FixableCVECount from 'Components/FixableCVECount';
import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';

const CVEStackedPill = ({ horizontal, vulnCounter, hideLink, url, fixableUrl }) => {
    const { critical, high, medium, low, all } = vulnCounter;
    const tooltipBody = (
        <div>
            <div>
                {critical.total} Critical CVEs ({critical.fixable} Fixable)
            </div>
            <div>
                {high.total} High CVEs ({high.fixable} Fixable)
            </div>
            <div>
                {medium.total} Medium CVEs ({medium.fixable} Fixable)
            </div>
            <div>
                {low.total} Low CVEs ({low.fixable} Fixable)
            </div>
        </div>
    );

    const tooltip = { title: 'Criticality Distribution', body: tooltipBody };
    const width = horizontal ? '' : 'w-16';
    return (
        <div className="flex items-center">
            <div className={`mr-4 ${width}`}>
                <FixableCVECount
                    cves={all.total}
                    fixable={all.fixable}
                    orientation={horizontal ? 'horizontal' : 'vertical'}
                    url={url}
                    fixableUrl={fixableUrl}
                    hideLink={hideLink}
                />
            </div>
            <SeverityStackedPill
                critical={critical.total}
                high={high.total}
                medium={medium.total}
                low={low.total}
                tooltip={tooltip}
            />
        </div>
    );
};

CVEStackedPill.propTypes = {
    horizontal: PropTypes.bool,
    vulnCounter: PropTypes.shape({
        critical: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number
        }),
        high: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number
        }),
        medium: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number
        }),
        low: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number
        }),
        all: PropTypes.shape({
            total: PropTypes.number,
            fixable: PropTypes.number
        })
    }).isRequired,
    hideLink: PropTypes.bool,
    url: PropTypes.string,
    fixableUrl: PropTypes.string
};

CVEStackedPill.defaultProps = {
    horizontal: false,
    hideLink: false,
    url: '',
    fixableUrl: ''
};

export default CVEStackedPill;
