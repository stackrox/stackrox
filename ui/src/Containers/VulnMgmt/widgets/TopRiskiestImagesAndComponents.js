import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import { format } from 'date-fns';

import workflowStateContext from 'Containers/workflowStateContext';
import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import NoResultsMessage from 'Components/NoResultsMessage';
import TextSelect from 'Components/TextSelect';
import Widget from 'Components/Widget';
import CVEStackedPill from 'Components/CVEStackedPill';
import NumberedList from 'Components/NumberedList';
import NoComponentVulnMessage from 'Components/NoComponentVulnMessage';
import queryService from 'modules/queryService';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';
import { entitySortFieldsMap } from 'constants/sortFields';

const TOP_RISKIEST_IMAGES = gql`
    query topRiskiestImages($query: String, $pagination: Pagination) {
        results: images(query: $query, pagination: $pagination) {
            id
            name {
                fullName
            }
            vulnCounter {
                all {
                    total
                    fixable
                }
                low {
                    total
                    fixable
                }
                medium {
                    total
                    fixable
                }
                high {
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

const TOP_RISKIEST_COMPONENTS = gql`
    query topRiskiestComponents($query: String, $pagination: Pagination) {
        results: components(query: $query, pagination: $pagination) {
            id
            name
            version
            lastScanned
            vulnCounter {
                all {
                    total
                    fixable
                }
                low {
                    total
                    fixable
                }
                medium {
                    total
                    fixable
                }
                high {
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
    const results = data.results
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
            const cveListState = newState.pushList(entityTypes.CVE);
            const cvesUrl = cveListState.toUrl();
            const fixableUrl = cveListState.setSearch({ Fixable: true }).toUrl();

            const { critical, high, medium, low } = vulnCounter;

            const scanTimeToUse = scan?.scanTime || lastScanned;
            const formattedDate = format(scanTimeToUse, dateTimeFormat);
            const tooltipTitle =
                formattedDate && formattedDate !== 'Invalid Date'
                    ? formattedDate
                    : 'Date and time not available';
            const tooltipBody = (
                <div className="flex-1 border-base-300 overflow-hidden">
                    <div className="mb-2">
                        <span className="text-base-600 font-700 mr-2">
                            {entityType === entityTypes.IMAGE ? 'Image:' : 'Component:'}
                        </span>
                        <span className="font-600">{text}</span>
                    </div>
                    <div>
                        <span className="text-base-600 font-700 mr-2 mb-1">
                            Criticality Distribution:
                        </span>
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
                    body: tooltipBody
                }
            };
        });

    return results;
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
    if (entityContext === {} || !entityContext[entityTypes.COMPONENT]) {
        entities.push({ label: 'Top Riskiest Components', value: entityTypes.COMPONENT });
    }
    if (entityContext === {} || !entityContext[entityTypes.IMAGE] || entities.length === 0) {
        // unshift so it sits at the front of the list (in case both entity types are added, image should come first)
        entities.unshift({ label: 'Top Riskiest Images', value: entityTypes.IMAGE });
    }
    return entities;
};

const TopRiskiestImagesAndComponents = ({ entityContext, limit }) => {
    const entities = getEntitiesByContext(entityContext);

    const [selectedEntity, setSelectedEntity] = useState(entities[0].value);

    function onEntityChange(value) {
        setSelectedEntity(value);
    }

    const { loading, data = {}, error } = useQuery(getQueryBySelectedEntity(selectedEntity), {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
            pagination: queryService.getPagination(
                {
                    id: 'Priority',
                    desc: false
                },
                WIDGET_PAGINATION_START_OFFSET,
                limit
            )
        }
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
            const entityText = selectedEntity === entityTypes.COMPONENT ? 'components' : 'images';
            content = (
                <NoResultsMessage
                    message={`An error occurred in retrieving ${entityText}. Please refresh the page. If this problem continues, please contact support.`}
                    className="p-3"
                    icon="warn"
                />
            );
        } else if (data && data.results && data.results === 0) {
            content = (
                <div className="flex mx-auto items-center">No scanner setup for this registry.</div>
            );
        } else {
            const processedData = processData(data, selectedEntity, workflowState, limit);

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

TopRiskiestImagesAndComponents.propTypes = {
    entityContext: PropTypes.shape({}),
    limit: PropTypes.number
};

TopRiskiestImagesAndComponents.defaultProps = {
    entityContext: {},
    limit: 5
};

export default TopRiskiestImagesAndComponents;
