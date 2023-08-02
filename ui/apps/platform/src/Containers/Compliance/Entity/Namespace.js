import React, { useContext, useState } from 'react';
import pluralize from 'pluralize';

import ComplianceByStandards from 'Containers/Compliance/widgets/ComplianceByStandards';
import Query from 'Components/CacheFirstQuery';
import IconWidget from 'Components/IconWidget';
import CountWidget from 'Components/CountWidget';
import Cluster from 'images/cluster.svg';
import { NAMESPACE_QUERY as QUERY } from 'queries/namespace';
import Widget from 'Components/Widget';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable-next-line import/no-cycle */
import ComplianceList from 'Containers/Compliance/List/List';
import ResourceCount from 'Containers/Compliance/widgets/ResourceCount';
import PageNotFound from 'Components/PageNotFound';
import isGQLLoading from 'utils/gqlLoading';
import Loader from 'Components/Loader';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import Labels from 'Containers/Compliance/widgets/Labels';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';

import Header from './Header';
import ResourceTabs from './ResourceTabs';

function processData(data, entityId) {
    const defaultValue = {
        labels: [],
        name: '',
        clusterName: '',
        id: entityId,
    };

    if (!data || !data.results || !data.results.metadata) {
        return defaultValue;
    }

    const { metadata, ...rest } = data.results;

    return {
        ...rest,
        ...metadata,
    };
}

const NamespacePage = ({
    entityId,
    listEntityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query,
    sidePanelMode,
}) => {
    const [isExporting, setIsExporting] = useState(false);
    const searchParam = useContext(searchContext);
    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }
                if (!data.results) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.NAMESPACE}
                            useCase={useCases.COMPLIANCE}
                        />
                    );
                }
                const namespace = processData(data);
                const { name, id, clusterName, labels, numNetworkPolicies } = namespace;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                if (listEntityType1 && !sidePanelMode) {
                    const listQuery = {
                        groupBy:
                            listEntityType1 === entityTypes.CONTROL ? entityTypes.STANDARD : '',
                        Namespace: namespace?.name,
                        ...query[searchParam],
                    };
                    contents = (
                        <section
                            id="capture-list"
                            className="flex flex-col flex-1 overflow-y-auto h-full"
                        >
                            <ComplianceList
                                entityType={listEntityType1}
                                query={listQuery}
                                selectedRowId={entityId1}
                                entityType2={entityType2}
                                entityListType2={entityListType2}
                                entityId2={entityId2}
                            />
                        </section>
                    );
                } else {
                    contents = (
                        <div
                            className={`flex-1 relative bg-base-200 overflow-auto ${
                                !sidePanelMode ? `p-6` : `p-4`
                            } `}
                            id="capture-dashboard"
                        >
                            <div
                                className={`grid ${
                                    !sidePanelMode
                                        ? `grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense`
                                        : ``
                                } sm:grid-columns-1 grid-gap-5`}
                            >
                                <div
                                    className={`grid s-2 md:grid-auto-fit md:grid-dense ${pdfClassName}`}
                                    style={{ '--min-tile-width': '50%' }}
                                >
                                    <div className="s-full pb-3">
                                        <EntityCompliance
                                            entityType={entityTypes.NAMESPACE}
                                            entityId={id}
                                            entityName={name}
                                            clusterName={clusterName}
                                        />
                                    </div>
                                    <div className="md:pr-3 pt-3">
                                        <IconWidget
                                            title="Parent Cluster"
                                            icon={Cluster}
                                            description={clusterName}
                                            loading={loading}
                                        />
                                    </div>
                                    <div className="md:pl-3 pt-3">
                                        <CountWidget
                                            title="Network Policies"
                                            count={numNetworkPolicies}
                                        />
                                    </div>
                                </div>

                                <Widget
                                    className={`sx-2 ${pdfClassName}`}
                                    header={`${labels.length} ${pluralize('Label', labels.length)}`}
                                >
                                    <Labels labels={labels} />
                                </Widget>
                                <ComplianceByStandards
                                    entityId={id}
                                    entityName={name}
                                    entityType={entityTypes.NAMESPACE}
                                />
                                {sidePanelMode && (
                                    <>
                                        <div
                                            className={`grid sx-2 sy-1 md:grid-auto-fit md:grid-dense ${pdfClassName}`}
                                            style={{ '--min-tile-width': '50%' }}
                                        >
                                            <div className="md:pr-3 pt-3">
                                                <ResourceCount
                                                    entityType={entityTypes.DEPLOYMENT}
                                                    relatedToResourceType={entityTypes.NAMESPACE}
                                                    relatedToResource={namespace}
                                                />
                                            </div>
                                            <div className="md:pl-3 pt-3">
                                                <ResourceCount
                                                    entityType={entityTypes.SECRET}
                                                    relatedToResourceType={entityTypes.NAMESPACE}
                                                    relatedToResource={namespace}
                                                />
                                            </div>
                                        </div>
                                    </>
                                )}
                            </div>
                        </div>
                    );
                }

                return (
                    <section className="flex flex-col h-full w-full">
                        {!sidePanelMode && (
                            <>
                                <Header
                                    entityType={entityTypes.NAMESPACE}
                                    listEntityType={listEntityType1}
                                    entityName={name}
                                    entityId={id}
                                    isExporting={isExporting}
                                    setIsExporting={setIsExporting}
                                />
                                <ResourceTabs
                                    entityId={id}
                                    entityType={entityTypes.NAMESPACE}
                                    selectedType={listEntityType1}
                                    resourceTabs={[entityTypes.CONTROL, entityTypes.DEPLOYMENT]}
                                />
                            </>
                        )}
                        {contents}
                        {isExporting && <BackdropExporting />}
                    </section>
                );
            }}
        </Query>
    );
};
NamespacePage.propTypes = entityPagePropTypes;
NamespacePage.defaultProps = entityPageDefaultProps;

export default NamespacePage;
