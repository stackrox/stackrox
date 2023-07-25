import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
import { format } from 'date-fns';

import workflowStateContext from 'Containers/workflowStateContext';
import Loader from 'Components/Loader';
import NoResultsMessage from 'Components/NoResultsMessage';
import TextSelect from 'Components/TextSelect';
import Widget from 'Components/Widget';
import CVEStackedPill from 'Components/CVEStackedPill';
import NumberedList from 'Components/NumberedList';
import NoComponentVulnMessage from 'Components/NoComponentVulnMessage';
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import queryService from 'utils/queryService';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';
import { entitySortFieldsMap } from 'constants/sortFields';
import { entityPriorityField } from '../VulnMgmt.constants';
import {
    entityNounOrdinaryCasePlural,
    entityNounSentenceCaseSingular,
} from '../entitiesForVulnerabilityManagement';

import ViewAllButton from './ViewAllButton';

const TOP_RISKIEST_IMAGE_VULNS = gql`
    query topRiskiestImageVulns($query: String, $pagination: Pagination) {
        results: images(query: $query, pagination: $pagination) {
            id
            name {
                fullName
            }
            vulnCounter: imageVulnerabilityCounter {
                all {
                    total
                    fixable
                }
                low {
                    total
                    fixable
                }
                moderate {
                    total
                    fixable
                }
                important {
                    total
                    fixable
                }
                critical {
                    total
                    fixable
                }
            }
            priority
            scan {
                scanTime
            }
        }
    }
`;

const TOP_RISKIEST_IMAGE_COMPONENTS = gql`
    query topRiskiestImageComponents($query: String, $pagination: Pagination) {
        results: imageComponents(query: $query, pagination: $pagination) {
            id
            name
            version
            lastScanned
            vulnCounter: imageVulnerabilityCounter {
                all {
                    total
                    fixable
                }
                low {
                    total
                    fixable
                }
                moderate {
                    total
                    fixable
                }
                important {
                    total
                    fixable
                }
                critical {
                    total
                    fixable
                }
            }
            priority
        }
    }
`;

const TOP_RISKIEST_NODE_COMPONENTS = gql`
    query topRiskiestNodeComponents($query: String, $pagination: Pagination) {
        results: nodeComponents(query: $query, pagination: $pagination) {
            id
            name
            version
            lastScanned
            vulnCounter: nodeVulnerabilityCounter {
                all {
                    total
                    fixable
                }
                low {
                    total
                    fixable
                }
                moderate {
                    total
                    fixable
                }
                important {
                    total
                    fixable
                }
                critical {
                    total
                    fixable
                }
            }
            priority
        }
    }
`;

const TOP_RISKIEST_NODE_VULNS = gql`
    query topRiskiestNodeVulns($query: String, $pagination: Pagination) {
        results: nodes(query: $query, pagination: $pagination) {
            id
            name
            vulnCounter: nodeVulnerabilityCounter {
                all {
                    total
                    fixable
                }
                low {
                    total
                    fixable
                }
                moderate {
                    total
                    fixable
                }
                important {
                    total
                    fixable
                }
                critical {
                    total
                    fixable
                }
            }
            priority
            scan {
                scanTime
            }
        }
    }
`;

const getTextByEntityType = (entityType, data) => {
    switch (entityType) {
        case entityTypes.NODE:
        case entityTypes.NODE_CVE:
            return data.name.fullName || data.name;
        case entityTypes.COMPONENT:
        case entityTypes.NODE_COMPONENT:
        case entityTypes.IMAGE_COMPONENT:
            return `${data.name}:${data.version}`;
        case entityTypes.IMAGE:
        case entityTypes.IMAGE_CVE:
        default:
            return data.name.fullName;
    }
};

function getQueryBySelectedEntityVulns(entityType) {
    switch (entityType) {
        case entityTypes.IMAGE_COMPONENT:
            return TOP_RISKIEST_IMAGE_COMPONENTS;
        case entityTypes.NODE_COMPONENT:
            return TOP_RISKIEST_NODE_COMPONENTS;
        case entityTypes.NODE:
            return TOP_RISKIEST_NODE_VULNS;
        case entityTypes.IMAGE:
        default:
            return TOP_RISKIEST_IMAGE_VULNS;
    }
}

const getEntitiesByContext = (entityContext) => {
    const entities = [];
    if (!entityContext[entityTypes.NODE_COMPONENT] && !entityContext[entityTypes.IMAGE]) {
        entities.push({
            label: 'Top riskiest node components',
            value: entityTypes.NODE_COMPONENT,
        });
    }
    if (!entityContext[entityTypes.IMAGE_COMPONENT] && !entityContext[entityTypes.NODE]) {
        entities.push({
            label: 'Top riskiest image components',
            value: entityTypes.IMAGE_COMPONENT,
        });
    }
    if (
        (!entityContext[entityTypes.IMAGE] && !entityContext[entityTypes.NODE]) ||
        entities.length === 0
    ) {
        // unshift so it sits at the front of the list (in case both entity types are added, image should come first)
        entities.unshift({
            label: 'Top riskiest images',
            value: entityTypes.IMAGE,
        });
    }
    if (!entityContext[entityTypes.NODE] && !entityContext[entityTypes.IMAGE]) {
        entities.push({
            label: 'Top riskiest nodes',
            value: entityTypes.NODE,
        });
    }
    return entities;
};

function getCVEListType(entityType) {
    switch (entityType) {
        case entityTypes.NODE:
        case entityTypes.NODE_COMPONENT:
            return entityTypes.NODE_CVE;
        case entityTypes.IMAGE:
        case entityTypes.IMAGE_COMPONENT:
        default:
            return entityTypes.IMAGE_CVE;
    }
}

const processData = (data, entityType, workflowState) => {
    const results = data.results
        .slice()
        .sort((a, b) => {
            const d = a.priority - b.priority;
            if (d === 0) {
                return d;
            }
            if (a.priority === 0) {
                return 1;
            }
            if (b.priority === 0) {
                return -1;
            }
            return d;
        })
        .map(({ id, vulnCounter, scan, lastScanned, ...rest }) => {
            const text = getTextByEntityType(entityType, { ...rest });
            const newState = workflowState.pushRelatedEntity(entityType, id);

            const url = newState.toUrl();
            const cveListType = getCVEListType(entityType);
            const cveListState = newState.pushList(cveListType);
            const cvesUrl = cveListState.toUrl();
            const fixableUrl = cveListState.setSearch({ Fixable: true }).toUrl();

            const { critical, important, moderate, low } = vulnCounter;

            const scanTimeToUse = scan?.scanTime || lastScanned;
            const formattedDate = format(scanTimeToUse, dateTimeFormat);
            const tooltipTitle =
                formattedDate && formattedDate !== 'Invalid Date'
                    ? formattedDate
                    : 'Date and time not available';
            const tooltipBody = (
                <div className="flex-1 border-base-300 overflow-hidden">
                    <div className="mb-2">
                        <span className="font-700 mr-2 capitalize">
                            {entityNounSentenceCaseSingular[entityType]}:
                        </span>
                        <span>{text}</span>
                    </div>
                    <div>
                        <span className="font-700 mr-2 mb-1">Criticality Distribution:</span>
                        <div>
                            {critical.total} Critical CVEs ({critical.fixable} Fixable)
                        </div>
                        <div>
                            {important.total} Important CVEs ({important.fixable} Fixable)
                        </div>
                        <div>
                            {moderate.total} Moderate CVEs ({moderate.fixable} Fixable)
                        </div>
                        <div>
                            {low.total} Low CVEs ({low.fixable} Fixable)
                        </div>
                    </div>
                </div>
            );

            return {
                text,
                url,
                component: (
                    <div className="flex">
                        <CVEStackedPill
                            vulnCounter={vulnCounter}
                            url={cvesUrl}
                            fixableUrl={fixableUrl}
                            horizontal
                            showTooltip={false}
                        />
                    </div>
                ),
                tooltip: {
                    title: tooltipTitle,
                    body: tooltipBody,
                },
            };
        });

    return results;
};

const TopRiskiestEntities = ({ entityContext, search, limit }) => {
    const entities = getEntitiesByContext(entityContext);
    const [selectedEntity, setSelectedEntity] = useState(entities[0].value);

    function onEntityChange(value) {
        setSelectedEntity(value);
    }

    const entityContextObject = queryService.entityContextToQueryObject(entityContext); // deals with BE inconsistency

    const queryObject = {
        ...entityContextObject,
        ...search,
    }; // Combine entity context and search
    const query = queryService.objectToWhereClause(queryObject); // get final gql query string

    const {
        loading,
        data = {},
        error,
    } = useQuery(getQueryBySelectedEntityVulns(selectedEntity), {
        variables: {
            query,
            pagination: queryService.getPagination(
                {
                    id: entityPriorityField[selectedEntity],
                    desc: false,
                },
                WIDGET_PAGINATION_START_OFFSET,
                limit
            ),
        },
    });

    const workflowState = useContext(workflowStateContext);

    const viewAllURL = workflowState
        .pushList(selectedEntity)
        .setSort([{ id: entitySortFieldsMap[selectedEntity].PRIORITY, desc: false }])
        .toUrl();

    const headerComponents = <ViewAllButton url={viewAllURL} />;

    let content = <Loader />;

    if (!loading) {
        if (error) {
            const defaultMessage = `An error occurred in retrieving ${entityNounOrdinaryCasePlural[selectedEntity]}. Please refresh the page. If this problem continues, please contact support.`;

            const parsedMessage = checkForPermissionErrorMessage(error, defaultMessage);

            content = <NoResultsMessage message={parsedMessage} className="p-3" icon="warn" />;
        } else if (data && data.results && data.results === 0) {
            content = (
                <div className="flex mx-auto items-center">No scanner setup for this registry.</div>
            );
        } else {
            const processedData = processData(data, selectedEntity, workflowState);

            if (processedData.length) {
                content = (
                    <div className="w-full">
                        <NumberedList data={processedData} linkLeftOnly />
                    </div>
                );
            } else {
                content = <NoComponentVulnMessage />;
            }
        }
    }

    return (
        <Widget
            className="h-full pdf-page"
            titleComponents={
                <TextSelect value={selectedEntity} options={entities} onChange={onEntityChange} />
            }
            headerComponents={headerComponents}
        >
            {content}
        </Widget>
    );
};

TopRiskiestEntities.propTypes = {
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    limit: PropTypes.number,
};

TopRiskiestEntities.defaultProps = {
    entityContext: {},
    search: {},
    limit: 5,
};

export default TopRiskiestEntities;
