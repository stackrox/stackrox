import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';

import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import ViewAllButton from 'Components/ViewAllButton';
import Sunburst from 'Components/visuals/Sunburst';
import NoComponentVulnMessage from 'Components/NoComponentVulnMessage';
import workflowStateContext from 'Containers/workflowStateContext';
import {
    severityColorMap,
    severityTextColorMap,
    severityColorLegend
} from 'constants/severityColors';

const CVES_QUERY = gql`
    query getCvesByCVSS($query: String, $scopeQuery: String) {
        results: vulnerabilities(query: $query) {
            cve
            cvss
            isFixable(query: $scopeQuery)
            severity
            summary
        }
    }
`;

const CvesByCvssScore = ({ entityContext }) => {
    const { loading, data = {} } = useQuery(CVES_QUERY, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
            scopeQuery: queryService.entityContextToQueryString(entityContext)
        }
    });

    let content = <Loader />;
    let header;

    const workflowState = useContext(workflowStateContext);
    const viewAllURL = workflowState
        .pushList(entityTypes.CVE)
        .setSort([{ id: 'cvss', desc: true }])
        .toUrl();

    function getChildren(vulns, severity) {
        return vulns
            .filter(vuln => vuln.severity === severity)
            .map(({ cve, cvss, summary }) => {
                const severityString = `${severity.toUpperCase()}_SEVERITY`;
                return {
                    severity,
                    name: `${cve} -- ${summary}`,
                    color: severityColorMap[severityString],
                    labelColor: severityTextColorMap[severityString],
                    textColor: severityTextColorMap[severityString],
                    value: cvss,
                    link: workflowState.pushRelatedEntity(entityTypes.CVE, cve).toUrl()
                };
            });
    }

    function getSunburstData(vulns) {
        return severityColorLegend.map(({ title, color, textColor }) => {
            const severity = title.toUpperCase();
            return {
                name: title,
                color,
                children: getChildren(vulns, severity),
                textColor,
                value: 0
            };
        });
    }

    function getSidePanelData(vulns) {
        return severityColorLegend.map(({ title, textColor }) => {
            const severity = title.toUpperCase();
            const category = vulns.filter(vuln => vuln.severity === severity);
            const text = `${category.length} rated as ${title}`;
            return {
                text,
                textColor
            };
        });
    }
    if (!loading) {
        if (!data || !data.results) {
            content = (
                <div className="flex mx-auto items-center">No scanner setup for this registry.</div>
            );
        } else if (!data.results.length) {
            content = <NoComponentVulnMessage />;
        } else {
            const sunburstData = getSunburstData(data.results);
            const sidePanelData = getSidePanelData(data.results).reverse();
            header = <ViewAllButton url={viewAllURL} />;
            content = (
                <Sunburst
                    data={sunburstData}
                    rootData={sidePanelData}
                    totalValue={data.results.length}
                    units="value"
                    small
                />
            );
        }
    }

    return (
        <Widget className="h-full pdf-page" header="CVEs by CVSS Score" headerComponents={header}>
            {content}
        </Widget>
    );
};

CvesByCvssScore.propTypes = {
    entityContext: PropTypes.shape({})
};

CvesByCvssScore.defaultProps = {
    entityContext: {}
};

export default CvesByCvssScore;
