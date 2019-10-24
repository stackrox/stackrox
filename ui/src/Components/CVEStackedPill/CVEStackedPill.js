import React from 'react';
import PropTypes from 'prop-types';
import FixableCVECount from 'Components/FixableCVECount';
import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';

const CVEStackedPill = ({ horizontal, vulnCounter, pdf, url }) => {
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
    return (
        <div className="flex items-center w-full">
            <div className="mr-4">
                <FixableCVECount
                    cves={all.total}
                    fixable={all.fixable}
                    orientation={horizontal ? 'horizontal' : 'vertical'}
                    url={url}
                    pdf={pdf}
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
    pdf: PropTypes.bool,
    url: PropTypes.string
};

CVEStackedPill.defaultProps = {
    horizontal: false,
    pdf: false,
    url: ''
};

export default CVEStackedPill;
