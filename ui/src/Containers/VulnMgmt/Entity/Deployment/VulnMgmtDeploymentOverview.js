import React, { useContext } from 'react';
import { format } from 'date-fns';

import CollapsibleSection from 'Components/CollapsibleSection';
import Metadata from 'Components/Metadata';
import RelatedEntity from 'Components/RelatedEntity';
import RelatedEntityListCount from 'Components/RelatedEntityListCount';
import EntityList from 'Components/EntityList';
import entityTypes from 'constants/entityTypes';
import {
    getCveTableColumns,
    renderCveDescription,
    defaultCveSort
} from 'Containers/VulnMgmt/List/Cves/VulnMgmtListCves';
import { getDefaultExpandedRows } from 'Containers/Workflow/WorkflowListPage';
import workflowStateContext from 'Containers/workflowStateContext';
import dateTimeFormat from 'constants/dateTimeFormat';

const VulnMgmtDeploymentOverview = ({ data, entityContext }) => {
    const workflowState = useContext(workflowStateContext);

    const {
        cluster,
        created,
        type,
        replicas,
        labels = [],
        annotations = [],
        namespace,
        namespaceId,
        serviceAccount,
        serviceAccountID,
        imageCount,
        secretCount,
        vulnerabilities
    } = data;

    const metadataKeyValuePairs = [
        {
            key: 'Created',
            value: created ? format(created, dateTimeFormat) : 'N/A'
        },
        {
            key: 'Deployment Type',
            value: type
        },
        {
            key: 'Replicas',
            value: replicas
        }
    ];

    const cveTableColumns = getCveTableColumns(workflowState);
    // TODO: move filtering to the GraphQL query, if it becomes available at that level
    const fixableCves = vulnerabilities.filter(vuln => vuln.isFixable);
    const expandedCveRows = getDefaultExpandedRows(fixableCves);

    return (
        <div className="w-full" id="capture-dashboard-stretch">
            <CollapsibleSection title="Deployment Details">
                <div className="flex mb-4 flex-wrap pdf-page">
                    <Metadata
                        className="mx-4 bg-base-100 h-48 mb-4"
                        keyValuePairs={metadataKeyValuePairs}
                        labels={labels}
                        annotations={annotations}
                    />
                    {!entityContext.CLUSTER && cluster && (
                        <RelatedEntity
                            className="mx-4 min-w-48 h-48 mb-4"
                            entityType={entityTypes.CLUSTER}
                            entityId={cluster.id}
                            name="Cluster"
                            value={cluster.name}
                        />
                    )}
                    {!entityContext.NAMESPACE && (
                        <RelatedEntity
                            className="mx-4 min-w-48 h-48 mb-4"
                            entityType={entityTypes.NAMESPACE}
                            entityId={namespaceId}
                            name="Namespace"
                            value={namespace}
                        />
                    )}
                    {!entityContext.SERVICE_ACCOUNT && (
                        <RelatedEntity
                            className="mx-4 min-w-48 h-48 mb-4"
                            entityType={entityTypes.SERVICE_ACCOUNT}
                            name="Service Account"
                            value={serviceAccount}
                            entityId={serviceAccountID}
                        />
                    )}
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Images"
                        value={imageCount}
                        entityType={entityTypes.IMAGE}
                    />
                    <RelatedEntityListCount
                        className="mx-4 min-w-48 h-48 mb-4"
                        name="Secrets"
                        value={secretCount}
                        entityType={entityTypes.SECRET}
                    />
                </div>
            </CollapsibleSection>
            <CollapsibleSection title="Deployment Findings">
                <EntityList
                    entityType={entityTypes.CVE}
                    idAttribute="cve"
                    rowData={fixableCves}
                    tableColumns={cveTableColumns}
                    selectedRowId={null}
                    search={null}
                    SubComponent={renderCveDescription}
                    defaultSorted={defaultCveSort}
                    defaultExpanded={expandedCveRows}
                />
            </CollapsibleSection>
        </div>
    );
};

export default VulnMgmtDeploymentOverview;
