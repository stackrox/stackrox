import React from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

import SeverityTile from 'Containers/Dashboard/SeverityTile';
import { severityColorMap } from 'constants/severityColors';
import { useTheme } from 'Containers/ThemeProvider';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import severityPropType from './severityPropTypes';

const EnvironmentRisk = ({ globalViolationsCounts, clusters }) => {
    const { isDarkMode } = useTheme();
    const counts = {
        CRITICAL_SEVERITY: 0,
        HIGH_SEVERITY: 0,
        MEDIUM_SEVERITY: 0,
        LOW_SEVERITY: 0,
    };
    globalViolationsCounts.forEach((group) => {
        group.counts.forEach((d) => {
            const count = parseInt(d.count, 10);
            counts[d.severity] += count;
        });
    });

    const severities = Object.keys(counts);
    const totalViolations = Object.values(counts).reduce((a, b) => a + b);
    return (
        <div className="w-full">
            <h2
                className={`-ml-6 inline-block leading-normal mb-6 px-3 pl-6 pr-4 rounded-r-full text-base-600 text-lg text-primary-800 tracking-wide tracking-widest uppercase ${
                    !isDarkMode ? 'bg-base-100' : 'bg-base-0'
                }`}
            >
                <Link
                    className="text-base-600 hover:text-primary-600 flex items-center h-10"
                    to="/main/violations"
                >
                    <span>
                        {totalViolations === 1
                            ? `${totalViolations} System Violation`
                            : `${totalViolations} System Violations`}
                    </span>
                </Link>
            </h2>
            <div className="flex">
                {severities.map((severity, i) => {
                    const searchFilter = {
                        Severity: severity,
                        Cluster: clusters,
                    };
                    const searchString = getUrlQueryStringForSearchFilter(searchFilter);
                    const link = `/main/violations?${searchString}`;
                    return (
                        <SeverityTile
                            severity={severity}
                            count={counts[severity]}
                            color={severityColorMap[severity]}
                            index={i}
                            key={severity}
                            link={link}
                        />
                    );
                })}
            </div>
        </div>
    );
};

EnvironmentRisk.propTypes = {
    globalViolationsCounts: PropTypes.arrayOf(
        PropTypes.shape({
            counts: PropTypes.arrayOf(
                PropTypes.shape({
                    count: PropTypes.string.isRequired,
                    severity: severityPropType,
                })
            ),
            group: PropTypes.string.isRequired,
        })
    ).isRequired,
    clusters: PropTypes.arrayOf(PropTypes.string),
};

EnvironmentRisk.defaultProps = {
    clusters: [],
};

export default EnvironmentRisk;
