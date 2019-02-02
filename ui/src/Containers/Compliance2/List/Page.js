import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import AppLink from 'Components/AppLink';

import Panel from 'Components/Panel';
import CollapsibleBanner from 'Components/CollapsibleBanner/CollapsibleBanner';
import ComplianceEntityPage from 'Containers/Compliance2/Entity/Page';
import pageTypes from 'constants/pageTypes';
import SearchInput from './SearchInput';
import Header from './Header';
import ListTable from './Table';

class ComplianceListPage extends Component {
    static propTypes = {
        match: ReactRouterPropTypes.match.isRequired,
        location: ReactRouterPropTypes.location.isRequired,
        params: PropTypes.shape({})
    };

    static defaultProps = {
        params: null
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0,
            selectedRow: null
        };
    }

    updateSelectedRow = selectedRow => this.setState({ selectedRow });

    clearSelectedRow = () => {
        this.setState({ selectedRow: null });
    };

    renderSidePanel = () => {
        const { selectedRow } = this.state;
        if (!selectedRow) return '';

        const { match, location } = this.props;
        const { context, query, entityType } = URLService.getParams(match, location);

        const pageParams = {
            context,
            pageType: pageTypes.ENTITY,
            entityType,
            entityId: selectedRow.control
        };

        const linkParams = {
            query,
            entityId: selectedRow.id,
            entityType
        };

        const headerTextComponent = (
            <AppLink
                context={context}
                pageType={pageTypes.ENTITY}
                entityType={entityType}
                params={linkParams}
            >
                <div
                    className="flex flex-1 text-base-600 uppercase items-center tracking-wide pl-4 pt-1 leading-normal font-700"
                    data-test-id="panel-header"
                >
                    {selectedRow.id}
                </div>
            </AppLink>
        );

        return (
            <Panel
                className="w-2/3"
                headerTextComponent={headerTextComponent}
                onClose={this.clearSelectedRow}
            >
                <ComplianceEntityPage params={pageParams} sidePanelMode />
            </Panel>
        );
    };

    setTablePage = page => this.setState({ page });

    render() {
        const { match, location } = this.props;
        const params = URLService.getParams(match, location);
        const { selectedRow, page } = this.state;
        return (
            <section className="flex flex-col h-full">
                <Header searchComponent={<SearchInput />} />
                <CollapsibleBanner>
                    <div>widgets here</div>
                </CollapsibleBanner>
                <div className="flex flex-1 overflow-y-auto">
                    <ListTable
                        selectedRow={selectedRow}
                        page={page}
                        params={params}
                        updateSelectedRow={this.updateSelectedRow}
                        setTablePage={this.setTablePage}
                    />
                    {this.renderSidePanel()}
                </div>
            </section>
        );
    }
}

export default withRouter(ComplianceListPage);
