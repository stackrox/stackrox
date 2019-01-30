import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { horizontalBarData, sunburstData, sunburstLegendData } from 'mockData/graphDataMock';
import entityTypes, { standardEntityTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';

import Panel from 'Components/Panel';
import Widget from 'Components/Widget';
import CollapsibleBanner from 'Components/CollapsibleBanner/CollapsibleBanner';
import StandardsAcrossEntity from 'Containers/Compliance2/widgets/StandardsAcrossEntity';
import StandardsByEntity from 'Containers/Compliance2/widgets/StandardsByEntity';
import Sunburst from 'Components/visuals/Sunburst';
import ComplianceEntityPage from 'Containers/Compliance2/Entity/Page';
import pageTypes from 'constants/pageTypes';
import SearchInput from './SearchInput';
import Header from './Header';
import ListTable from './Table';

// Ultimately, this will need to be dynamic
const entity = standardEntityTypes.CONTROL;

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
        const pageParams = URLService.getParams(match, location);

        const compliancePageParams = {
            context: pageParams.context,
            pageType: pageTypes.ENTITY,
            entityType: entity,
            entityId: selectedRow.control
        };

        return (
            <Panel
                className="w-2/3"
                header={selectedRow.node || selectedRow.control}
                onClose={this.clearSelectedRow}
            >
                <ComplianceEntityPage params={compliancePageParams} sidePanelMode />
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
                    <StandardsAcrossEntity type={entityTypes.CLUSTERS} data={horizontalBarData} />
                    <StandardsByEntity type={entityTypes.CLUSTERS} />
                    <Widget header="Compliance Across Controls">
                        <Sunburst
                            data={sunburstData}
                            legendData={sunburstLegendData}
                            centerLabel="75%"
                            containerProps={{
                                style: {
                                    borderRight: '1px solid var(--base-300)'
                                }
                            }}
                        />
                    </Widget>
                    <StandardsAcrossEntity type={entityTypes.NAMESPACES} data={horizontalBarData} />
                    <StandardsAcrossEntity type={entityTypes.NODES} data={horizontalBarData} />
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
