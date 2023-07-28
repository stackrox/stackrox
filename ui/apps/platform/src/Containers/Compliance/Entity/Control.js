import React, { useContext, useState } from 'react';

import entityTypes from 'constants/entityTypes';
import Widget from 'Components/Widget';
import Query from 'Components/CacheFirstQuery';
import { CONTROL_QUERY as QUERY } from 'queries/controls';
import ControlDetails from 'Components/ControlDetails';
import ControlRelatedResourceList from 'Containers/Compliance/widgets/ControlRelatedResourceList';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable-next-line import/no-cycle */
import ComplianceList from 'Containers/Compliance/List/List';
import Loader from 'Components/Loader';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import PageNotFound from 'Components/PageNotFound';
import searchContext from 'Containers/searchContext';
import isGQLLoading from 'utils/gqlLoading';

import Header from './Header';
import ResourceTabs from './ResourceTabs';

const ControlPage = ({
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
            {({ data, loading }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }

                if (!data || !data.results) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.CONTROL}
                            useCase={useCases.COMPLIANCE}
                        />
                    );
                }

                const { results: control, complianceStandards: standards } = data;
                const standard = standards.find((item) => item.id === control.standardId);
                const { name, standardId, interpretationText, description } = control;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                const standardName = standard ? standard.name : '';
                let contents;

                if (listEntityType1 && !sidePanelMode) {
                    const listQuery = {
                        control: name,
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
                                className={pdfClassName}
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
                                }sm:grid-columns-1 grid-gap-5`}
                            >
                                <ControlDetails
                                    standardId={standardId}
                                    standardName={standardName}
                                    control={name}
                                    description={description}
                                    className={`sx-2 ${pdfClassName}`}
                                />
                                {!!interpretationText.length && (
                                    <Widget
                                        className={`sx-2 ${pdfClassName}`}
                                        header="Control guidance"
                                    >
                                        <div className="p-4 leading-loose whitespace-pre-wrap">
                                            {interpretationText}
                                        </div>
                                    </Widget>
                                )}
                                {sidePanelMode && (
                                    <>
                                        <ControlRelatedResourceList
                                            listEntityType={entityTypes.CLUSTER}
                                            pageEntityType={entityTypes.CONTROL}
                                            pageEntity={control}
                                            standard={standardName}
                                            className={pdfClassName}
                                        />
                                        <ControlRelatedResourceList
                                            listEntityType={entityTypes.NAMESPACE}
                                            pageEntityType={entityTypes.CONTROL}
                                            pageEntity={control}
                                            standard={standardName}
                                            className={pdfClassName}
                                        />
                                        <ControlRelatedResourceList
                                            listEntityType={entityTypes.NODE}
                                            pageEntityType={entityTypes.CONTROL}
                                            pageEntity={control}
                                            standard={standardName}
                                            className={pdfClassName}
                                        />
                                        <ControlRelatedResourceList
                                            listEntityType={entityTypes.DEPLOYMENT}
                                            pageEntityType={entityTypes.CONTROL}
                                            pageEntity={control}
                                            standard={standardName}
                                            className={pdfClassName}
                                        />
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
                                    entityType={entityTypes.CONTROL}
                                    listEntityType={listEntityType1}
                                    entity={control}
                                    entityName={`${standardName} ${name}`}
                                    isExporting={isExporting}
                                    setIsExporting={setIsExporting}
                                />
                                <ResourceTabs
                                    entityId={entityId}
                                    entityType={entityTypes.CONTROL}
                                    selectedType={listEntityType1}
                                    standardId={standardId}
                                    resourceTabs={[
                                        entityTypes.NODE,
                                        entityTypes.DEPLOYMENT,
                                        entityTypes.CLUSTER,
                                    ]}
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
ControlPage.propTypes = entityPagePropTypes;
ControlPage.defaultProps = entityPageDefaultProps;

export default ControlPage;
