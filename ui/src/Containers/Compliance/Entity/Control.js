import React from 'react';
import PropTypes from 'prop-types';
import isEmpty from 'lodash/isEmpty';
import standardLabels from 'messages/standards';
import entityTypes from 'constants/entityTypes';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import { CONTROL_QUERY as QUERY } from 'queries/controls';
import ControlDetails from 'Containers/Compliance/widgets/ControlDetails';
import ControlRelatedResourceList from 'Containers/Compliance/widgets/ControlRelatedResourceList';
import Header from './Header';

const ControlPage = ({ sidePanelMode, params }) => (
    <Query query={QUERY} variables={{ id: params.entityId }} pollInterval={5000}>
        {({ data }) => {
            const { results: control, complianceStandards: standards } = data;
            if (isEmpty(control)) return null;
            const standard = standards.find(item => item.id === control.standardId);
            const { name, standardId, interpretationText, description } = control;
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
                                className="s-2"
                            />
                            {!!interpretationText.length && (
                                <Widget className="sx-2" header="Control guidance">
                                    <div className="p-4 leading-loose">{interpretationText}</div>
                                </Widget>
                            )}
                            {!sidePanelMode && (
                                <>
                                    <ControlRelatedResourceList
                                        listEntityType={entityTypes.CLUSTER}
                                        pageEntityType={entityTypes.CONTROL}
                                        pageEntity={control}
                                        standard={standard ? standard.name : ''}
                                    />
                                    <ControlRelatedResourceList
                                        listEntityType={entityTypes.NAMESPACE}
                                        pageEntityType={entityTypes.CONTROL}
                                        pageEntity={control}
                                        standard={standard ? standard.name : ''}
                                    />
                                    <ControlRelatedResourceList
                                        listEntityType={entityTypes.NODE}
                                        pageEntityType={entityTypes.CONTROL}
                                        pageEntity={control}
                                        standard={standard ? standard.name : ''}
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

ControlPage.propTypes = {
    sidePanelMode: PropTypes.bool,
    params: PropTypes.shape({}).isRequired
};

ControlPage.defaultProps = {
    sidePanelMode: false
};

export default ControlPage;
