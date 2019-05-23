import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { standardBaseTypes, standardEntityTypes } from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import ListTable from './Table';
import SidePanel from './SidePanel';
import EvidenceControlList from './EvidenceControlList';
import EvidenceResourceList from './EvidenceResourceList';

class ComplianceList extends Component {
    static propTypes = {
        searchComponent: PropTypes.node,
        entityType: PropTypes.string.isRequired,
        query: PropTypes.shape({}),
        match: ReactRouterPropTypes.match.isRequired,
        location: ReactRouterPropTypes.location.isRequired
    };

    static defaultProps = {
        searchComponent: null,
        query: null
    };

    constructor(props) {
        super(props);
        this.state = {
            selectedRow: null
        };
    }

    componentWillReceiveProps(nextProps) {
        if (nextProps.location !== this.props.location) {
            this.setState({ selectedRow: null });
        }
    }

    updateSelectedRow = selectedRow => this.setState({ selectedRow });

    clearSelectedRow = () => {
        this.setState({ selectedRow: null });
    };

    getContentsForControlsList = () => {
        const { selectedRow } = this.state;
        const { searchComponent, entityType } = this.props;

        let sidePanel = null;

        if (selectedRow) {
            const {
                control: { name, id: selectedId, name: control, standardId }
            } = selectedRow;
            const linkText = control ? `${standardBaseTypes[standardId]} ${control}` : name;
            sidePanel = (
                <SidePanel
                    entityType={entityType}
                    entityId={selectedId}
                    clearSelectedRow={this.clearSelectedRow}
                    linkText={linkText}
                    standardId={standardId}
                    controlResult={selectedRow}
                />
            );
        }

        return (
            <div className="flex flex-1">
                <EvidenceControlList
                    searchComponent={searchComponent}
                    selectedRow={selectedRow}
                    updateSelectedRow={this.updateSelectedRow}
                    resourceType={entityType}
                />
                {sidePanel}
            </div>
        );
    };

    getContentsForEvidenceResourceList = () => {
        const { selectedRow } = this.state;
        const { searchComponent, entityType } = this.props;

        let sidePanel = null;

        if (selectedRow) {
            const {
                control: { standardId },
                resource: { id: selectedId, name: resourceName }
            } = selectedRow;
            const linkText = resourceName;
            sidePanel = (
                <SidePanel
                    entityType={entityType}
                    entityId={selectedId}
                    clearSelectedRow={this.clearSelectedRow}
                    linkText={linkText}
                    standardId={standardId}
                    controlResult={selectedRow}
                />
            );
        }

        return (
            <div className="flex flex-1">
                <EvidenceResourceList
                    searchComponent={searchComponent}
                    resourceType={entityType}
                    selectedRow={selectedRow}
                    updateSelectedRow={this.updateSelectedRow}
                />
                {sidePanel}
            </div>
        );
    };

    getContentsForComplianceList = () => {
        const { selectedRow } = this.state;
        const { searchComponent, entityType, query } = this.props;

        let sidePanel = null;

        if (selectedRow) {
            const { name, id: selectedId, control, standardId } = selectedRow;
            const linkText = control ? `${standardBaseTypes[standardId]} ${control}` : name;
            sidePanel = (
                <SidePanel
                    entityType={entityType}
                    entityId={selectedId}
                    clearSelectedRow={this.clearSelectedRow}
                    linkText={linkText}
                    standardId={standardId}
                />
            );
        }

        return (
            <div className="flex w-full">
                <ListTable
                    searchComponent={searchComponent}
                    selectedRow={selectedRow}
                    entityType={entityType}
                    query={query}
                    updateSelectedRow={this.updateSelectedRow}
                    pdfId="capture-list"
                />
                {sidePanel}
            </div>
        );
    };

    render() {
        const { entityType, match, location } = this.props;
        let contents;

        const { entityType: pageType } = URLService.getParams(match, location);

        if (entityType === standardEntityTypes.CONTROL) {
            contents = this.getContentsForControlsList();
        } else if (pageType === standardEntityTypes.CONTROL) {
            contents = this.getContentsForEvidenceResourceList();
        } else {
            contents = this.getContentsForComplianceList();
        }

        return <div className="flex flex-1 overflow-y-auto h-full bg-base-100">{contents}</div>;
    }
}

export default withRouter(ComplianceList);
