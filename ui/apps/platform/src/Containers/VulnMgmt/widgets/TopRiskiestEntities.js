import React, { useState, useContext } from 'react';
import PropTypes from 'prop-types';
import { gql, useQuery } from '@apollo/client';
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
import { checkForPermissionErrorMessage } from 'utils/permissionUtils';
import queryService from 'utils/queryService';
import dateTimeFormat from 'constants/dateTimeFormat';
import entityTypes from 'constants/entityTypes';
import { WIDGET_PAGINATION_START_OFFSET } from 'constants/workflowPages.constants';
import { entitySortFieldsMap } from 'constants/sortFields';
import { resourceLabels } from 'messages/common';
import { entityPriorityField } from 'Containers/VulnMgmt/VulnMgmt.constants';
import useFeatureFlags from 'hooks/useFeatureFlags';

// TODO: remove once ROX_FRONTEND_VM_UDPATES is enabled
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

const TOP_RISKIEST_NODES = gql`
    query topRiskiestNodes($query: String, $pagination: Pagination) {
        results: nodes(query: $query, pagination: $pagination) {
            id
            name
            vulnCounter {
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
            return data.name;
        case entityTypes.COMPONENT:
            return `${data.name}:${data.version}`;
        case entityTypes.IMAGE:
        default:
            return data.name.fullName;
    }
};

function getSelectedEntity(selectedEntity, showVmUpdates) {
    if (!showVmUpdates) {
        return selectedEntity;
    }
    switch (selectedEntity) {
        case entityTypes.NODE:
            return entityTypes.NODE_CVE;
        case entityTypes.IMAGE:
        default:
            return entityTypes.IMAGE_CVE;
    }
}

const processData = (data, entityType, workflowState, showVmUpdates) => {
    const currentEntityType = getSelectedEntity(entityType, showVmUpdates);
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
            const text = getTextByEntityType(currentEntityType, { ...rest });
            const newState = workflowState.pushRelatedEntity(currentEntityType, id);

            const url = newState.toUrl();
            const cveListState = newState.pushList(currentEntityType);
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
                        <span className="text-base-600 font-700 mr-2 capitalize">
                            {resourceLabels[currentEntityType]}:
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

const getQueryBySelectedEntity = (entityType) => {
    switch (entityType) {
        case entityTypes.COMPONENT:
            return TOP_RISKIEST_COMPONENTS;
        case entityTypes.NODE:
            return TOP_RISKIEST_NODES;
        case entityTypes.IMAGE:
        default:
            return TOP_RISKIEST_IMAGES;
    }
};

const getEntitiesByContext = (entityContext, showVmUpdates) => {
    const entities = [];
    if (!showVmUpdates && (entityContext === {} || !entityContext[entityTypes.COMPONENT])) {
        entities.push({
            label: 'Top Riskiest Components',
            value: entityTypes.COMPONENT,
        });
    }
    if (entityContext === {} || !entityContext[entityTypes.IMAGE] || entities.length === 0) {
        // unshift so it sits at the front of the list (in case both entity types are added, image should come first)
        entities.unshift({
            label: `Top Riskiest Image${showVmUpdates ? ' Vulnerabilities' : ''}`,
            value: showVmUpdates ? entityTypes.IMAGE_CVE : entityTypes.IMAGE,
        });
    }
    if (entityContext === {} || !entityContext[entityTypes.NODE]) {
        entities.push({
            label: `Top Riskiest Node${showVmUpdates ? ' Vulnerabilities' : ''}`,
            value: showVmUpdates ? entityTypes.NODE_CVE : entityTypes.NODE,
        });
    }
    return entities;
};

const TopRiskiestEntities = ({ entityContext, limit }) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVmUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UDPATES');
    const entities = getEntitiesByContext(entityContext, showVmUpdates);
    const [selectedEntity, setSelectedEntity] = useState(entities[0].value);

    function onEntityChange(value) {
        setSelectedEntity(value);
    }

    const {
        loading,
        data = {},
        error,
    } = useQuery(getQueryBySelectedEntity(selectedEntity), {
        variables: {
            query: queryService.entityContextToQueryString(entityContext),
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
        .pushList(getSelectedEntity(selectedEntity, showVmUpdates))
        .setSort([{ id: entitySortFieldsMap[selectedEntity].PRIORITY, desc: false }])
        .toUrl();

    const headerComponents = <ViewAllButton url={viewAllURL} />;

    let content = <Loader />;

    if (!loading) {
        if (error) {
            const defaultMessage = `An error occurred in retrieving ${resourceLabels[selectedEntity]}s. Please refresh the page. If this problem continues, please contact support.`;

            const parsedMessage = checkForPermissionErrorMessage(error, defaultMessage);

            content = <NoResultsMessage message={parsedMessage} className="p-3" icon="warn" />;
        } else if (data && data.results && data.results === 0) {
            content = (
                <div className="flex mx-auto items-center">No scanner setup for this registry.</div>
            );
        } else {
            const processedData = processData(data, selectedEntity, workflowState, showVmUpdates);

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
    limit: PropTypes.number,
};

TopRiskiestEntities.defaultProps = {
    entityContext: {},
    limit: 5,
};

export default TopRiskiestEntities;
