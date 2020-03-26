import gql from 'graphql-tag';

import logError from 'modules/logError';
import ANALYST_NOTES_TYPES from 'constants/analystnotes';

export const GET_ALERT_COMMENTS = gql`
    query getAlertComments($resourceId: ID!) {
        comments: alertComments(resourceId: $resourceId) {
            resourceType
            resourceId
            user {
                email
                id
                name
            }
            id: commentId
            message: commentMessage
            createdTime: createdAt
            updatedTime: lastModified
            isModifiable: modifiable
        }
    }
`;

export const GET_PROCESS_COMMENTS = gql`
    query getProcessComments(
        $deploymentID: ID!
        $containerName: String!
        $execFilePath: String!
        $args: String!
    ) {
        comments: processComments(
            deploymentID: $deploymentID
            containerName: $containerName
            execFilePath: $execFilePath
            args: $args
        ) {
            resourceType
            resourceId
            user {
                email
                id
                name
            }
            id: commentId
            message: commentMessage
            createdTime: createdAt
            updatedTime: lastModified
            isModifiable: modifiable
        }
    }
`;

export const ADD_ALERT_COMMENT = gql`
    mutation addAlertComment($resourceId: ID!, $commentMessage: String!) {
        addAlertComment(resourceId: $resourceId, commentMessage: $commentMessage)
    }
`;

export const ADD_PROCESS_COMMENT = gql`
    mutation addProcessComment(
        $deploymentID: ID!
        $containerName: String!
        $execFilePath: String!
        $args: String!
        $commentMessage: String!
    ) {
        addProcessComment(
            deploymentID: $deploymentID
            containerName: $containerName
            execFilePath: $execFilePath
            args: $args
            commentMessage: $commentMessage
        )
    }
`;

export const UPDATE_ALERT_COMMENT = gql`
    mutation updateAlertComment($resourceId: ID!, $commentId: ID!, $commentMessage: String!) {
        updateAlertComment(
            resourceId: $resourceId
            commentId: $commentId
            commentMessage: $commentMessage
        )
    }
`;

export const UPDATE_PROCESS_COMMENT = gql`
    mutation updateProcessComment(
        $deploymentID: ID!
        $containerName: String!
        $execFilePath: String!
        $args: String!
        $commentId: ID!
        $commentMessage: String!
    ) {
        updateProcessComment(
            deploymentID: $deploymentID
            containerName: $containerName
            execFilePath: $execFilePath
            args: $args
            commentId: $commentId
            commentMessage: $commentMessage
        )
    }
`;

export const REMOVE_ALERT_COMMENT = gql`
    mutation removeAlertComment($resourceId: ID!, $commentId: ID!) {
        removeAlertComment(resourceId: $resourceId, commentId: $commentId)
    }
`;

export const REMOVE_PROCESS_COMMENT = gql`
    mutation removeProcessComment(
        $deploymentID: ID!
        $containerName: String!
        $execFilePath: String!
        $args: String!
        $commentId: ID!
    ) {
        removeProcessComment(
            deploymentID: $deploymentID
            containerName: $containerName
            execFilePath: $execFilePath
            args: $args
            commentId: $commentId
        )
    }
`;

/**
 * @typedef {Object} Result
 * @property {string} GET_COMMENTS - The GraphQL query used to fetch comments
 * @property {string} ADD_COMMENT - The GraphQL query used to add a comment
 * @property {string} UPDATE_COMMENT - The GraphQL query used to update a comment
 * @property {string} REMOVE_COMMENT - The GraphQL query used to remove a comment
 */

/**
 * Returns the queries used for fetching, adding, updating, and removing comments
 * based on the type
 * @param {string} type - The tags type (ie. VIOLATION and PROCESS)
 * @returns {Result} - returns an object with queries
 */
export const getQueriesByType = type => {
    if (type === ANALYST_NOTES_TYPES.VIOLATION) {
        return {
            GET_COMMENTS: GET_ALERT_COMMENTS,
            ADD_COMMENT: ADD_ALERT_COMMENT,
            UPDATE_COMMENT: UPDATE_ALERT_COMMENT,
            REMOVE_COMMENT: REMOVE_ALERT_COMMENT
        };
    }
    if (type === ANALYST_NOTES_TYPES.PROCESS) {
        return {
            GET_COMMENTS: GET_PROCESS_COMMENTS,
            ADD_COMMENT: ADD_PROCESS_COMMENT,
            UPDATE_COMMENT: UPDATE_PROCESS_COMMENT,
            REMOVE_COMMENT: REMOVE_PROCESS_COMMENT
        };
    }
    const error = `Queries for type (${type}) do not exist`;
    logError(new Error(error));
    return {};
};
