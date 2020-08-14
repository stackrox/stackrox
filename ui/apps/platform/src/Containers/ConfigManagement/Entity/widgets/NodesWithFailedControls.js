import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import NoResultsMessage from 'Components/NoResultsMessage';
import { gql, useQuery } from '@apollo/client';
import Raven from 'raven-js';
import queryService from 'utils/queryService';
import { entityAcrossControlsColumns } from 'constants/listColumns';
import uniqBy from 'lodash/uniqBy';

import Loader from 'Components/Loader';
import TableWidget from './TableWidget';

const NODES_WITH_FAILING_CONTROLS = gql`
    query nodesWithFailingControls($query: String) {
        executedControls(query: $query) {
            complianceControl {
                id
                name
                complianceControlFailingNodes {
                    id
                    name
                    clusterName
                }
                complianceControlPassingNodes {
                    id
                    name
                    clusterName
                }
            }
            controlStatus
        }
    }
`;

const filterByEntityContext = (entityContext) => {
    const result = Object.keys(entityContext).reduce((acc, entityType) => {
        const entityId = entityContext[entityType];
        acc[`${entityType} Id`] = entityId;
        return acc;
    }, {});
    return queryService.objectToWhereClause(result);
};

const getFailingNodes = (executedControls) => {
    const failingNodes = executedControls.reduce((acc, curr) => {
        return [...acc, ...curr.complianceControl.complianceControlFailingNodes];
    }, []);
    return uniqBy(failingNodes, 'id').map((node) => ({ ...node, passing: false }));
};

const getPassingNodes = (executedControls) => {
    const passingNodes = executedControls.reduce((acc, curr) => {
        return [...acc, ...curr.complianceControl.complianceControlPassingNodes];
    }, []);
    return uniqBy(passingNodes, 'id').map((node) => ({ ...node, passing: true }));
};

const NodesWithFailedControls = (props) => {
    const { entityType, entityContext } = props;
    const { loading, error, data } = useQuery(NODES_WITH_FAILING_CONTROLS, {
        variables: {
            query: filterByEntityContext(entityContext),
        },
        fetchPolicy: 'no-cache',
    });
    if (loading) {
        return (
            <div className="flex flex-1 items-center justify-center p-6">
                <Loader />
            </div>
        );
    }
    if (error) {
        Raven.captureException(error);
    }
    if (!data) {
        return null;
    }
    const { executedControls = [] } = data;
    if (executedControls.length === 0) {
        return (
            <NoResultsMessage
                message={`No nodes failing ${
                    entityType === entityTypes.CONTROL ? 'this control' : 'any controls'
                }`}
                className="p-6"
                icon="info"
            />
        );
    }

    const failingNodes = getFailingNodes(executedControls);
    const passingNodes = getPassingNodes(executedControls);
    const numFailing = failingNodes.length;
    const numPassing = passingNodes.length;
    if (numPassing && !numFailing) {
        return (
            <NoResultsMessage
                message={`No nodes failing ${
                    entityType === entityTypes.CONTROL ? 'this control' : 'any controls'
                }`}
                className="p-3 shadow"
                icon="info"
            />
        );
    }
    if (!numPassing && !numFailing) {
        return (
            <NoResultsMessage
                message={`Findings ${
                    entityContext[entityTypes.CONTROL] ? 'for this control' : 'across controls'
                } could not be assessed`}
                className="p-3 shadow"
                icon="warn"
            />
        );
    }
    const tableHeader = `${numFailing} ${numFailing === 1 ? 'node is' : 'nodes are'} ${
        entityType === entityTypes.CONTROL ? 'failing this control' : 'failing controls'
    }`;
    return (
        <TableWidget
            entityType={entityTypes.NODE}
            header={tableHeader}
            rows={failingNodes}
            noDataText="No Nodes"
            className="bg-base-100 w-full"
            columns={entityAcrossControlsColumns[entityTypes.NODE]}
            idAttribute="id"
            defaultSorted={[
                {
                    id: 'name',
                    desc: false,
                },
            ]}
        />
    );
};

NodesWithFailedControls.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityContext: PropTypes.shape({}).isRequired,
};

export default NodesWithFailedControls;
