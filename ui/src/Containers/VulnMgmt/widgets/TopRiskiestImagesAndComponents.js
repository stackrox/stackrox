import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import sortBy from 'lodash/sortBy';

import workflowStateContext from 'Containers/workflowStateContext';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import TextSelect from 'Components/TextSelect';
import Widget from 'Components/Widget';
import CVEStackedPill from 'Components/CVEStackedPill';
import NumberedList from 'Components/NumberedList';
import NoComponentVulnMessage from 'Components/NoComponentVulnMessage';

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
        }
    }
`;

const TOP_RISKIEST_COMPONENTS = gql`
    query topRiskiestComponents($query: String) {
        results: components(query: $query) {
            id
            name
            version
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

const processData = (data, entityType, workflowState, limit) => {
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
        .map(({ id, vulnCounter, ...rest }) => {
            const text = getTextByEntityType(entityType, { ...rest });
            const newState = workflowState.pushRelatedEntity(entityType, id);

            const url = newState.toUrl();
            const cveListState = newState.pushList(entityTypes.CVE);
            const cvesUrl = cveListState.toUrl();
            const fixableUrl = cveListState.setSearch({ 'Fixed By': 'r/.*' }).toUrl();

            return {
                text,
                url,
                component: (
                    <CVEStackedPill
                        vulnCounter={vulnCounter}
                        url={cvesUrl}
                        fixableUrl={fixableUrl}
                        horizontal
                    />
                )
            };
        });
    const processedData = sortBy(results, [datum => datum.priority]).slice(0, limit); // @TODO: Remove when we have pagination on image components
    return processedData;
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

const TopRiskiestImagesAndComponents = ({ entityContext, limit }) => {
    const entities = getEntitiesByContext(entityContext);

    const [selectedEntity, setSelectedEntity] = useState(entities[0].value);

    function onEntityChange(value) {
        setSelectedEntity(value);
    }

    const { loading, data = {} } = useQuery(getQueryBySelectedEntity(selectedEntity), {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
            pagination: {
                /*
                limit
                @TODO: When priority is a sortable field, uncomment this

                sortOption: {
                    field: 'priority',
                    reversed: true
                }
            } */
            }
        }
    });

    let content = <Loader />;
    let headerComponents = null;

    const workflowState = useContext(workflowStateContext);

    if (!loading) {
        if (!data || !data.results) {
            content = (
                <div className="flex mx-auto items-center">No scanner setup for this registry.</div>
            );
        } else {
            const processedData = processData(data, selectedEntity, workflowState, limit);

            if (processedData.length) {
                content = (
                    <div className="w-full">
                        <NumberedList data={processedData} />
                    </div>
                );

                const viewAllURL = workflowState.pushList(selectedEntity).toUrl();
                headerComponents = <ViewAllButton url={viewAllURL} />;
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
