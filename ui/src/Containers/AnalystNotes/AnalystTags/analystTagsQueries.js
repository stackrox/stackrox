import gql from 'graphql-tag';
import logError from 'modules/logError';

import ANALYST_NOTES_TYPES from 'constants/analystnotes';

export const GET_ALERT_TAGS = gql`
    query getAlertTags($resourceId: ID!) {
        violation(id: $resourceId) {
            id
            tags
        }
    }
`;

export const GET_PROCESS_TAGS = gql`
    query getProcessTags(
        $deploymentID: ID!
        $containerName: String!
        $execFilePath: String!
        $args: String!
    ) {
        processTags(
            deploymentID: $deploymentID
            containerName: $containerName
            execFilePath: $execFilePath
            args: $args
        )
    }
`;

export const ADD_ALERT_TAGS = gql`
    mutation addAlertTags($resourceId: ID!, $tags: [String!]!) {
        addAlertTags(resourceId: $resourceId, tags: $tags)
    }
`;

export const ADD_PROCESS_TAGS = gql`
    mutation addProcessTags(
        $deploymentID: ID!
        $containerName: String!
        $execFilePath: String!
        $args: String!
        $tags: [String!]!
    ) {
        addProcessTags(
            deploymentID: $deploymentID
            containerName: $containerName
            execFilePath: $execFilePath
            args: $args
            tags: $tags
        )
    }
`;

export const REMOVE_ALERT_TAGS = gql`
    mutation removeAlertTags($resourceId: ID!, $tags: [String!]!) {
        removeAlertTags(resourceId: $resourceId, tags: $tags)
    }
`;

export const REMOVE_PROCESS_TAGS = gql`
    mutation removeProcessTags(
        $deploymentID: ID!
        $containerName: String!
        $execFilePath: String!
        $args: String!
        $tags: [String!]!
    ) {
        removeProcessTags(
            deploymentID: $deploymentID
            containerName: $containerName
            execFilePath: $execFilePath
            args: $args
            tags: $tags
        )
    }
`;

/**
 * Parses the API response data and returns the tags data
 * @param {string} type - The tags type (ie. VIOLATION and PROCESS)
 * @param {Object} data - The API response data
 * @returns {string[]} - returns the tags data from the API response
 */
export const getTagsDataByType = (type, data) => {
    if (!data) return [];
    if (type === ANALYST_NOTES_TYPES.VIOLATION) {
        return data.violation && data.violation.tags;
    }
    if (type === ANALYST_NOTES_TYPES.PROCESS) {
        return data.processTags;
    }
    const error = `Can't get data for type (${type}) because it does not exist`;
    logError(new Error(error));
    return [];
};

/**
 * @typedef {Object} Result
 * @property {string} GET_TAGS - The GraphQL query used to fetch tags
 * @property {string} ADD_TAGS - The GraphQL query used to add tags
 * @property {string} REMOVE_TAGS - The GraphQL query used to remove tags
 */

/**
 * Returns the queries used for fetching, adding, and removing tags based on the
 * type
 * @param {string} type - The tags type (ie. VIOLATION and PROCESS)
 * @returns {Result} - returns an object with queries
 */
export const getQueriesByType = type => {
    if (type === ANALYST_NOTES_TYPES.VIOLATION) {
        return {
            GET_TAGS: GET_ALERT_TAGS,
            ADD_TAGS: ADD_ALERT_TAGS,
            REMOVE_TAGS: REMOVE_ALERT_TAGS
        };
    }
    if (type === ANALYST_NOTES_TYPES.PROCESS) {
        return {
            GET_TAGS: GET_PROCESS_TAGS,
            ADD_TAGS: ADD_PROCESS_TAGS,
            REMOVE_TAGS: REMOVE_PROCESS_TAGS
        };
    }
    const error = `Queries for type (${type}) do not exist`;
    logError(new Error(error));
    return {};
};
