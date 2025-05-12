import React, { useContext } from 'react';
import { useLocation } from 'react-router-dom';
import { gql } from '@apollo/client';
import queryService from 'utils/queryService';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import ControlDetails from 'Components/ControlDetails';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import isGQLLoading from 'utils/gqlLoading';
import Widget from 'Components/Widget';
import searchContext from 'Containers/searchContext';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import NodesWithFailedControls from './widgets/NodesWithFailedControls';
import ConfigManagementListNodes from '../List/ConfigManagementListNodes';

const QUERY = gql`
    query getControl($id: ID!, $where: String) {
        results: complianceControl(id: $id) {
            interpretationText
            description
            id
            name
            standardId
            complianceControlNodes {
                name
                clusterName
                id
                clusterId
                osImage
                containerRuntimeVersion
                joinedAt
                nodeComplianceControlCount(query: $where) {
                    failingCount
                    passingCount
                    unknownCount
                }
            }
        }
    }
`;

const ConfigManagementEntityControl = ({ id, entityListType, query, entityContext }) => {
    const location = useLocation();
    const match = useWorkflowMatch();
    const searchParam = useContext(searchContext);

    const variables = {
        id,
        where: queryService.objectToWhereClause({
            ...query[searchParam],
            'Control Id': id,
        }),
    };

    return (
        <Query query={QUERY} variables={variables} fetchPolicy="network-only">
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }

                if (!data || !data.results) {
                    return <PageNotFound resourceType="CONTROL" useCase="configmanagement" />;
                }

                const { results: entity } = data;
                const { complianceControlNodes } = entity;

                if (entityListType) {
                    return (
                        <ConfigManagementListNodes
                            match={match}
                            location={location}
                            data={complianceControlNodes}
                            totalResults={complianceControlNodes?.length}
                            entityContext={{ ...entityContext, CONTROL: id }}
                        />
                    );
                }

                const {
                    standardId = '',
                    name = '',
                    description = '',
                    interpretationText = '',
                } = entity;

                return (
                    <div className="w-full" id="capture-dashboard-stretch">
                        <CollapsibleSection title="Control Summary">
                            <div className="flex flex-wrap pdf-page">
                                <ControlDetails
                                    standardId={standardId}
                                    control={name}
                                    description={description}
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                />
                                {!!interpretationText.length && (
                                    <Widget
                                        className="mx-4 min-w-48 min-h-48 mb-4 w-1/3 overflow-auto"
                                        header="Control guidance"
                                    >
                                        <div className="p-4 leading-loose whitespace-pre-wrap overflow-auto">
                                            {interpretationText}
                                        </div>
                                    </Widget>
                                )}
                                <RelatedEntityListCount
                                    className="mx-4 min-w-48 min-h-48 mb-4"
                                    name="Nodes"
                                    value={complianceControlNodes.length}
                                    entityType="NODE"
                                />
                            </div>
                        </CollapsibleSection>
                        {!(entityContext && entityContext.NODE) && (
                            <CollapsibleSection title="Control Findings">
                                <div className="flex pdf-page pdf-stretch shadow relative rounded bg-base-100 mb-4 ml-4 mr-4">
                                    <NodesWithFailedControls
                                        entityType="CONTROL"
                                        entityContext={{
                                            ...entityContext,
                                            CONTROL: id,
                                        }}
                                    />
                                </div>
                            </CollapsibleSection>
                        )}
                    </div>
                );
            }}
        </Query>
    );
};

ConfigManagementEntityControl.propTypes = entityComponentPropTypes;
ConfigManagementEntityControl.defaultProps = entityComponentDefaultProps;

export default ConfigManagementEntityControl;
