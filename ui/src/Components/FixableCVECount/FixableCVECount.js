import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const orientations = ['horizontal', 'vertical'];

const getOrientationClassName = orientation => {
    if (orientation === 'vertical') return '';
    return 'flex';
};

const FixableCVECount = ({ cves, fixable, url, orientation, pdf }) => {
    const className = `text-sm leading-normal whitespace-no-wrap ${getOrientationClassName(
        orientation
    )}`;
    let content = (
        <div className={className}>
            {!!cves && (
                <div className="text-primary-800 font-600 mx-1">
                    {cves} {cves.length === 1 ? 'CVE' : 'CVEs'}
                </div>
            )}
            {!!fixable && <div className="text-success-800 font-600">({fixable} Fixable)</div>}
        </div>
    );

    // This field is necessary to exclude rendering the Link during PDF generation. It causes an error where the Link can't be rendered outside a Router
    if (pdf) {
        return content;
    }
    if (url)
        content = (
            <Link to={url} className="w-full">
                {content}
            </Link>
        );
    return content;
};

FixableCVECount.propTypes = {
    cves: PropTypes.number,
    fixable: PropTypes.number,
    url: PropTypes.string,
    orientation: PropTypes.oneOf(orientations),
    pdf: PropTypes.bool
};

FixableCVECount.defaultProps = {
    cves: 0,
    fixable: 0,
    url: null,
    orientation: 'horizontal',
    pdf: false
};

export default FixableCVECount;
