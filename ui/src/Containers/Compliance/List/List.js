import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { standardBaseTypes, standardEntityTypes } from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import ListTable from './Table';
import SidePanel from './SidePanel';
import ControlsList from './ControlsList';

class ComplianceList extends Component {
    static propTypes = {
        searchComponent: PropTypes.node,
        entityType: PropTypes.string.isRequired,
        query: PropTypes.shape({}),
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
        const { entityType } = this.props;

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
                />
            );
        }

        return (
            <div className="flex flex-1">
                <ControlsList
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
            <div className="flex flex-1">
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
        const { entityType } = this.props;
        let contents;

        if (entityType === standardEntityTypes.CONTROL) {
            contents = this.getContentsForControlsList();
        } else {
            contents = this.getContentsForComplianceList();
        }

        return <div className="flex flex-1 overflow-y-auto h-full">{contents}</div>;
    }
}

export default withRouter(ComplianceList);
