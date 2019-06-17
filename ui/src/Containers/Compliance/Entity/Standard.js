import React from 'react';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import ComplianceList from 'Containers/Compliance/List/List';
import CollapsibleBanner from 'Components/CollapsibleBanner/CollapsibleBanner';
import ComplianceAcrossEntities from 'Containers/Compliance/widgets/ComplianceAcrossEntities';
import ControlsMostFailed from 'Containers/Compliance/widgets/ControlsMostFailed';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import SearchInput from '../SearchInput';
import Header from '../List/Header';

const StandardPage = ({
    entityId,
    listEntityType,
    entityId1,
    entityType2,
    entityListType2,
    entityId2
}) => {
    const listQuery = {
        'Standard Id': entityId
    };
    return (
        <section className="flex flex-col h-full relative" id="capture-list">
            <Header
                entityType={entityTypes.CONTROL}
                searchComponent={
                    <SearchInput categories={['COMPLIANCE']} shouldAddComplianceState />
                }
            />
            <CollapsibleBanner className="pdf-page">
                <ComplianceAcrossEntities entityType={entityTypes.CONTROL} query={listQuery} />
                <ControlsMostFailed entityType={entityTypes.CONTROL} query={listQuery} showEmpty />
            </CollapsibleBanner>
            <ComplianceList
                entityType={listEntityType}
                query={listQuery}
                selectedRowId={entityId1}
                entityType2={entityType2}
                entityListType2={entityListType2}
                entityId2={entityId2}
            />
        </section>
    );
};

StandardPage.propTypes = entityPagePropTypes;
StandardPage.defaultProps = entityPageDefaultProps;

export default withRouter(StandardPage);
