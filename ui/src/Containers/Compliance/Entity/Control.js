import React from 'react';
import PropTypes from 'prop-types';
import { standardLabels } from 'messages/standards';
import entityTypes, { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import { CONTROL_QUERY as QUERY } from 'queries/controls';
import ControlDetails from 'Containers/Compliance/widgets/ControlDetails';
import ControlRelatedResourceList from 'Containers/Compliance/widgets/ControlRelatedResourceList';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import ComplianceList from 'Containers/Compliance/List/List';
import Loader from 'Components/Loader';
import ResourceTabs from 'Components/ResourceTabs';
import ControlAssessment from 'Containers/Compliance/widgets/ControlAssessment';
import PageNotFound from 'Components/PageNotFound';
import Header from './Header';
import SearchInput from '../SearchInput';
import EntitiesAssessed from '../widgets/EntitiesAssessed';

const ControlPage = ({ match, location, controlId, sidePanelMode, controlResult }) => {
    const params = URLService.getParams(match, location);
    const entityId = controlId || params.controlId;
    const listEntityType = URLService.getEntityTypeKeyFromValue(params.listEntityType);

    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ data, loading }) => {
                if (loading) return <Loader />;
                if (!data || !data.results)
                    return <PageNotFound resourceType={entityTypes.CONTROL} />;

                const { results: control, complianceStandards: standards } = data;
                const standard = standards.find(item => item.id === control.standardId);
                const { name, standardId, interpretationText, description } = control;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                const standardName = standard ? standard.name : '';
                let contents;

                const searchComponent = listEntityType ? (
                    <SearchInput categories={[searchCategoryTypes[listEntityType]]} />
                ) : null;

                if (listEntityType && !sidePanelMode) {
                    const queryParams = { ...params.query };
                    queryParams.control = name;
                    const listQuery = {
                        ...queryParams
                    };
                    contents = (
                        <section
                            id="capture-list"
                            className="flex flex-col flex-1 overflow-y-auto h-full"
                        >
                            <ComplianceList
                                searchComponent={searchComponent}
                                entityType={listEntityType}
                                query={listQuery}
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
                                        <ControlAssessment
                                            className={`sx-2 ${pdfClassName}`}
                                            controlResult={controlResult}
                                        />
                                        <EntitiesAssessed
                                            className={`sx-2 ${pdfClassName}`}
                                            controlResult={controlResult}
                                        />
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
                                    listEntityType={null}
                                    entity={control}
                                    headerText={`${standardLabels[standardId]} ${name}`}
                                />
                                <ResourceTabs
                                    entityId={entityId}
                                    entityType={entityTypes.CONTROL}
                                    standardId={standardId}
                                    resourceTabs={[
                                        entityTypes.NODE,
                                        entityTypes.DEPLOYMENT,
                                        entityTypes.CLUSTER
                                    ]}
                                />
                            </>
                        )}
                        {contents}
                    </section>
                );
            }}
        </Query>
    );
};

ControlPage.propTypes = {
    sidePanelMode: PropTypes.bool,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    controlId: PropTypes.string,
    controlResult: PropTypes.shape({})
};

ControlPage.defaultProps = {
    sidePanelMode: false,
    controlId: null,
    controlResult: null
};

export default withRouter(ControlPage);
