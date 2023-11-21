/**
 * @jest-environment node
 *
 * Reference Error: TextEncoder is not defined
 * Maybe because of ReactDOMRenderer.renderToString in pdfUtils.js file.
 */

import entityTypes from 'constants/entityTypes';
import { getConfigMgmtCountQuery } from './ConfigMgmt.utils';

describe('ConfigMgmt.utils.test', () => {
    describe('getConfigMgmtCountQuery', () => {
        it('should return an empty string when the entity list is Controls', () => {
            const entityListType = entityTypes.CONTROL;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('');
        });

        it('should return an empty string when the entity list is Policies', () => {
            const entityListType = entityTypes.POLICY;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('');
        });

        it('should return an empty string when the entity list not a known type', () => {
            const entityListType = 'Wonka';

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('');
        });

        it('should return an appropriate count string when the entity list is Deployments', () => {
            const entityListType = entityTypes.DEPLOYMENT;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: deploymentCount(query: $query)');
        });

        it('should return an appropriate count string when the entity list is Images', () => {
            const entityListType = entityTypes.IMAGE;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: imageCount(query: $query)');
        });

        it('should return an appropriate count string when the entity list is Namespaces', () => {
            const entityListType = entityTypes.NAMESPACE;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: namespaceCount(query: $query)');
        });

        it('should return an appropriate count string when the entity list is Nodes', () => {
            const entityListType = entityTypes.NODE;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: nodeCount(query: $query)');
        });

        it('should return an appropriate count string when the entity list is Roles', () => {
            const entityListType = entityTypes.ROLE;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: k8sRoleCount(query: $query)');
        });

        it('should return an appropriate count string when the entity list is Secrets', () => {
            const entityListType = entityTypes.SECRET;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: secretCount(query: $query)');
        });

        it('should return an appropriate count string when the entity list is Service Accounts', () => {
            const entityListType = entityTypes.SERVICE_ACCOUNT;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: serviceAccountCount(query: $query)');
        });

        it('should return an appropriate count string when the entity list is Subjects', () => {
            const entityListType = entityTypes.SUBJECT;

            const countQuery = getConfigMgmtCountQuery(entityListType);

            expect(countQuery).toEqual('count: subjectCount(query: $query)');
        });
    });
});
