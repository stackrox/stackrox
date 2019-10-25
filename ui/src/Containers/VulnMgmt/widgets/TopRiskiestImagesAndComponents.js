import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';

import workflowStateContext from 'Containers/workflowStateContext';
import { generateURLTo } from 'modules/URLReadWrite';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import TextSelect from 'Components/TextSelect';
import Widget from 'Components/Widget';
import CVEStackedPill from 'Components/CVEStackedPill';
import NumberedList from 'Components/NumberedList';

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
        }
    }
`;

const TOP_RISKIEST_COMPONENTS = gql`
    query topRiskiestComponents($query: String) {
        results: imageComponents(query: $query) {
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
    const results = data.results.map(({ id, vulnCounter, ...rest }) => {
        const text = getTextByEntityType(entityType, { ...rest });
        const url = generateURLTo(workflowState, entityType, id);
        return {
            text,
            url,
            component: <CVEStackedPill vulnCounter={vulnCounter} url={url} horizontal />
        };
    });
    return results.splice(0, limit); // @TODO: Remove when we have pagination on image components
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
                limit
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
    const viewAllURL = generateURLTo(workflowState, entityTypes.IMAGE);
    if (!loading) {
        const processedData = processData(data, selectedEntity, workflowState, limit);

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
            headerComponents={<ViewAllButton url={viewAllURL} />}
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
    limit: 8
};

export default TopRiskiestImagesAndComponents;
