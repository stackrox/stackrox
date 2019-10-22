import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { Link } from 'react-router-dom';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import { getSeverityByCvss } from 'utils/vulnerabilityUtils';
import { severities } from 'constants/severities';
import queryService from 'modules/queryService';

import WorkflowStateMgr from 'modules/WorkflowStateManager';
import workflowStateContext from 'Containers/workflowStateContext';
import { generateURL } from 'modules/URLReadWrite';

import Button from 'Components/Button';
import Loader from 'Components/Loader';
import TextSelect from 'Components/TextSelect';
import Widget from 'Components/Widget';
import FixableCVECount from 'Components/FixableCVECount';
import SeverityStackedPill from 'Components/visuals/SeverityStackedPill';
import NumberedList from 'Components/NumberedList';

const TOP_RISKIEST_IMAGES = gql`
    query topRiskiestImages($query: String, $pagination: Pagination) {
        results: images(query: $query, pagination: $pagination) {
            id
            name {
                fullName
            }
            vulnCount
            vulns {
                cve
                isFixable
                cvss
            }
        }
    }
`;

const TOP_RISKIEST_COMPONENTS = gql`
    query topRiskiestComponents($query: String) {
        results: imageComponents(query: $query) {
            id
            name
            version
            vulnCount
            vulns {
                cve
                isFixable
                cvss
            }
        }
    }
`;

const ViewAllButton = ({ url }) => {
    return (
        <Link to={url} className="no-underline">
            <Button className="btn-sm btn-base" type="button" text="View All" />
        </Link>
    );
};

const getViewAllURL = workflowState => {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityTypes.IMAGE);
    const url = generateURL(workflowStateMgr.workflowState);
    return url;
};

const getSingleEntityURL = (workflowState, entityType, id) => {
    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.pushList(entityType).pushListItem(id);
    const url = generateURL(workflowStateMgr.workflowState);
    return url;
};

const getSeverityCountsAndTooltip = vulns => {
    const criticalCves = vulns.filter(
        vuln => getSeverityByCvss(vuln.cvss) === severities.CRITICAL_SEVERITY
    );
    const highCves = vulns.filter(
        vuln => getSeverityByCvss(vuln.cvss) === severities.HIGH_SEVERITY
    );
    const mediumCves = vulns.filter(
        vuln => getSeverityByCvss(vuln.cvss) === severities.MEDIUM_SEVERITY
    );
    const lowCves = vulns.filter(vuln => getSeverityByCvss(vuln.cvss) === severities.LOW_SEVERITY);

    const fixableCriticalCves = criticalCves.filter(vuln => !!vuln.isFixable);
    const fixableHighCves = highCves.filter(vuln => !!vuln.isFixable);
    const fixableMediumCves = mediumCves.filter(vuln => !!vuln.isFixable);
    const fixableLowCves = lowCves.filter(vuln => !!vuln.isFixable);

    const tooltip = {
        title: 'Criticality Distribution',
        body: (
            <div>
                <div>
                    {criticalCves.length} Critical CVES
                    {!!fixableCriticalCves.length && ` (${fixableCriticalCves.length} Fixable)`}
                </div>
                <div>
                    {highCves.length} High CVES
                    {!!fixableHighCves.length && ` (${fixableHighCves.length} Fixable)`}
                </div>
                <div>
                    {mediumCves.length} Medium CVEs
                    {!!fixableMediumCves.length && ` (${fixableMediumCves.length} Fixable)`}
                </div>
                <div>
                    {lowCves.length} Low CVES
                    {!!fixableLowCves.length && ` (${fixableLowCves.length} Fixable)`}
                </div>
            </div>
        )
    };

    return {
        critical: criticalCves.length,
        high: highCves.length,
        medium: mediumCves.length,
        low: lowCves.length,
        tooltip
    };
};

const getTextByEntityType = (entityType, data) => {
    switch (entityType) {
        case entityTypes.COMPONENT:
            return `${data.name}:${data.version}`;
        case entityTypes.IMAGE:
        default:
            return data.name.fullName;
    }
};

const processData = (data, entityType, workflowState) => {
    const results = data.results.map(({ id, vulnCount, vulns, ...rest }) => {
        const text = getTextByEntityType(entityType, { ...rest });
        const cvesCount = vulnCount;
        const fixableCvesCount = vulns.filter(vuln => vuln.isFixable).length;
        const { critical, high, medium, low, tooltip } = getSeverityCountsAndTooltip(vulns);
        return {
            text,
            url: getSingleEntityURL(workflowState, entityType, id),
            component: (
                <>
                    <div className="mr-4">
                        <FixableCVECount cves={cvesCount} fixable={fixableCvesCount} />
                    </div>
                    <SeverityStackedPill
                        critical={critical}
                        high={high}
                        medium={medium}
                        low={low}
                        tooltip={tooltip}
                    />
                </>
            )
        };
    });
    return results.splice(0, 8); // @TODO: Remove when we have pagination on image components
};

const getQueryBySelectedEntity = entityType => {
    switch (entityType) {
        case entityTypes.COMPONENT:
            return TOP_RISKIEST_COMPONENTS;
        case entityTypes.IMAGE:
        default:
            return TOP_RISKIEST_IMAGES;
    }
};

const getEntitiesByContext = entityContext => {
    const entities = [];
    if (entityContext === {} || !entityContext[entityTypes.IMAGE]) {
        entities.push({ label: 'Top Riskiest Images', value: entityTypes.IMAGE });
    }
    if (entityContext === {} || !entityContext[entityTypes.COMPONENT]) {
        entities.push({ label: 'Top Riskiest Components', value: entityTypes.COMPONENT });
    }
    return entities;
};

const TopRiskiestImagesAndComponents = ({ entityContext }) => {
    const entities = getEntitiesByContext(entityContext);

    const [selectedEntity, setSelectedEntity] = useState(entities[0].value);

    function onEntityChange(value) {
        setSelectedEntity(value);
    }

    const { loading, data = {} } = useQuery(getQueryBySelectedEntity(selectedEntity), {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
            pagination: {
                limit: 8
                /*
                @TODO: When priority is a sortable field, uncomment this

                sortOption: {
                    field: 'priority',
                    reversed: true
                }
                */
            }
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, selectedEntity, workflowState);

        content = (
            <div className="w-full">
                <NumberedList data={processedData} />
            </div>
        );
    }

    return (
        <Widget
            className="h-full pdf-page"
            titleComponents={
                <TextSelect value={selectedEntity} options={entities} onChange={onEntityChange} />
            }
            headerComponents={<ViewAllButton url={getViewAllURL(workflowState)} />}
        >
            {content}
        </Widget>
    );
};

TopRiskiestImagesAndComponents.propTypes = {
    entityContext: PropTypes.shape({})
};

TopRiskiestImagesAndComponents.defaultProps = {
    entityContext: {}
};

export default TopRiskiestImagesAndComponents;
