import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import * as Icon from 'react-feather';
import { severityLabels } from 'messages/common';
import { severityColorMap } from 'constants/severityColors';
import TwoLevelPieChart from 'Components/visuals/TwoLevelPieChart';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import severityPropType from './severityPropTypes';

const ViolationsByPolicyCategory = ({ data, history, clusters }) => {
    if (!data) {
        return '';
    }
    return data.map((policyType) => {
        const pieData = policyType.counts.map((d) => ({
            name: severityLabels[d.severity],
            value: parseInt(d.count, 10),
            color: severityColorMap[d.severity],
            onClick: () => {
                const searchFilter = {
                    Severity: d.severity,
                    Category: policyType.group,
                    Cluster: clusters,
                };
                const searchString = getUrlQueryStringForSearchFilter(searchFilter);
                history.push(`/main/violations?${searchString}`);
            },
        }));
        return (
            <div
                className="p-3 w-full lg:w-1/2 xl:w-1/3"
                key={policyType.group}
                data-testid="policy-category-violation"
            >
                <div className="bg-base-100 rounded-sm shadow h-full rounded">
                    <h2 className="flex items-center text-lg text-base font-sans text-base-600 tracking-wide border-base-300 border-b">
                        <Icon.BarChart className="h-4 w-4 m-3" />
                        <span className="px-4 py-4 pl-3 uppercase text-base tracking-wide pb-3 border-l border-base-300">
                            {policyType.group}
                        </span>
                    </h2>
                    <div className="m-4 h-64">
                        <TwoLevelPieChart data={pieData} />
                    </div>
                </div>
            </div>
        );
    });
};

ViolationsByPolicyCategory.propTypes = {
    data: PropTypes.arrayOf(
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
    history: PropTypes.shape({
        push: PropTypes.func.isRequired,
    }).isRequired,
    clusters: PropTypes.arrayOf(PropTypes.string),
};

ViolationsByPolicyCategory.defaultProps = {
    data: [],
    clusters: [],
};

export default withRouter(ViolationsByPolicyCategory);
