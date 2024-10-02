import React, { useContext, useState } from 'react';
import PropTypes from 'prop-types';
import { useLocation, useRouteMatch } from 'react-router-dom';
import lowerCase from 'lodash/lowerCase';
import pluralize from 'pluralize';

import URLService from 'utils/URLService';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import ComplianceList from 'Containers/Compliance/List/List';
import searchContext from 'Containers/searchContext';
import ComplianceSearchInput from '../ComplianceSearchInput';
import Header from './Header';

const ComplianceListPage = () => {
    const [isExporting, setIsExporting] = useState(false);
    const location = useLocation();
    const match = useRouteMatch();
    const params = URLService.getParams(match, location);
    const searchParam = useContext(searchContext);
    const query = { ...params.query[searchParam] };
    const { pageEntityListType, entityId1, entityType2, entityListType2, entityId2 } = params;
    const placeholder = `Filter ${pluralize(lowerCase(pageEntityListType))}`;
    return (
        <>
            <section className="flex flex-col h-full relative" id="capture-list">
                <Header
                    entityType={pageEntityListType}
                    searchComponent={
                        <ComplianceSearchInput
                            placeholder={placeholder}
                            categories={['COMPLIANCE']}
                            shouldAddComplianceState
                        />
                    }
                    standard={query.Standard || query.standard}
                    isExporting={isExporting}
                    setIsExporting={setIsExporting}
                />
                <ComplianceList
                    entityType={pageEntityListType}
                    query={query}
                    selectedRowId={entityId1}
                    entityType2={entityType2}
                    entityListType2={entityListType2}
                    entityId2={entityId2}
                    noSearch
                />
            </section>
            {isExporting && <BackdropExporting />}
        </>
    );
};

ComplianceListPage.propTypes = {
    params: PropTypes.shape({
        entityType: PropTypes.string.isRequired,
    }),
};

ComplianceListPage.defaultProps = {
    params: null,
};

export default ComplianceListPage;
