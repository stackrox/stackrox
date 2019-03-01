import React from 'react';
import PropTypes from 'prop-types';
import isEmpty from 'lodash/isEmpty';
import { standardLabels } from 'messages/standards';
import entityTypes from 'constants/entityTypes';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import { CONTROL_QUERY as QUERY } from 'queries/controls';
import ControlDetails from 'Containers/Compliance/widgets/ControlDetails';
import ControlRelatedResourceList from 'Containers/Compliance/widgets/ControlRelatedResourceList';
import URLService from 'modules/URLService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Header from './Header';

const ControlPage = ({ match, location, controlId, sidePanelMode }) => {
    const urlParams = URLService.getParams(match, location);

    return (
        <Query query={QUERY} variables={{ id: controlId || urlParams.controlId }}>
            {({ data }) => {
                const { results: control, complianceStandards: standards } = data;
                if (isEmpty(control)) return null;
                const standard = standards.find(item => item.id === control.standardId);
                const { name, standardId, interpretationText, description } = control;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                const standardName = standard ? standard.name : '';
                return (
                    <section className="flex flex-col h-full w-full">
                        {!sidePanelMode && (
                            <Header
                                header={`${standardLabels[standardId]} ${name}`}
                                subHeader="Control"
                            />
                        )}
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
                                        <div className="p-4 leading-loose">
                                            {interpretationText}
                                        </div>
                                    </Widget>
                                )}
                                {!sidePanelMode && (
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
                                    </>
                                )}
                            </div>
                        </div>
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
    controlId: PropTypes.string
};

ControlPage.defaultProps = {
    sidePanelMode: false,
    controlId: null
};

export default withRouter(ControlPage);
