import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

const orientations = ['horizontal', 'vertical'];

function getOrientationClassName(orientation) {
    return orientation === 'vertical' ? 'inline-flex flex-col' : 'inline-flex';
}

function stopPropagation(e) {
    e.stopPropagation();
}

const CountElement = ({ count, url, fixable, hideLink, individualClasses }) => {
    const classes = fixable
        ? 'text-success-700 hover:text-success-800 underline'
        : `text-base-600 ${url && !hideLink ? 'underline hover:text-primary-700' : ''}`;

    // can't just pluralize() because of special requirements
    //   1 CVE or 2 CVEs
    //   1 Fixable or 2 Fixable
    // @TODO: we could do this:
    //   const type = fixable ? 'Fixable' : pluralize('CVE', count);
    //
    //   after this bug in pluralize is fixed,
    //   https://github.com/blakeembrey/pluralize/issues/127
    const type = fixable ? 'Fixable' : 'CVE';
    const pluralized = count === 1 || fixable ? type : `${type}s`;

    const cveText = (
        <span className={`${classes} ${individualClasses}`}>{`${
            count === 0 ? 'No' : count
        } ${pluralized}`}</span>
    );
    const testId = fixable ? 'fixableCvesLink' : 'allCvesLink';

    return url && !hideLink ? (
        <Link to={url} onClick={stopPropagation} className="w-full" data-testid={testId}>
            {cveText}
        </Link>
    ) : (
        cveText
    );
};

const FixableCVECount = ({ cves, fixable, url, fixableUrl, orientation, hideLink, showZero }) => {
    const className = `text-sm items-center leading-normal whitespace-nowrap ${getOrientationClassName(
        orientation
    )}`;
    const individualClasses = orientation === 'horizontal' ? 'mr-1' : '';

    return (
        <div className={className}>
            {(showZero || !!cves) && (
                <CountElement
                    count={cves}
                    url={url}
                    hideLink={(showZero && !cves) || hideLink}
                    individualClasses={individualClasses}
                />
            )}
            {!!fixable && (
                <>
                    {` `}
                    <CountElement
                        count={fixable}
                        url={fixableUrl}
                        fixable
                        hideLink={(showZero && !cves) || hideLink}
                    />
                </>
            )}
        </div>
    );
};

FixableCVECount.propTypes = {
    cves: PropTypes.number,
    fixable: PropTypes.number,
    url: PropTypes.string,
    fixableUrl: PropTypes.string,
    orientation: PropTypes.oneOf(orientations),
    // This field is necessary to exclude rendering the Link during PDF generation. It causes an error where the Link can't be rendered outside a Router
    hideLink: PropTypes.bool,
    showZero: PropTypes.bool,
};

FixableCVECount.defaultProps = {
    cves: 0,
    fixable: 0,
    url: null,
    fixableUrl: null,
    orientation: 'horizontal',
    hideLink: false,
    showZero: false,
};

export default FixableCVECount;
