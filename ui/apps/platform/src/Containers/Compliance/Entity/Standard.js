import React from 'react';

import entityTypes from 'constants/entityTypes';
import ComplianceList from 'Containers/Compliance/List/List';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import SearchInput from '../SearchInput';
import Header from '../List/Header';

const StandardPage = ({
    entityId,
    listEntityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
}) => {
    const listQuery = {
        'Standard Id': entityId,
    };
    return (
        <section className="flex flex-col h-full relative" id="capture-list">
            <Header
                entityType={entityTypes.CONTROL}
                searchComponent={
                    <SearchInput categories={['COMPLIANCE']} shouldAddComplianceState />
                }
            />
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
};

StandardPage.propTypes = entityPagePropTypes;
StandardPage.defaultProps = entityPageDefaultProps;

export default StandardPage;
