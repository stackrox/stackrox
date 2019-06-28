import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';

import { ExternalLink as ExternalLinkIcon } from 'react-feather';
import Button from 'Components/Button';
import Panel from 'Components/Panel';
import EntityStage from './stages/EntityStage';
import RelatedEntityListStage from './stages/RelatedEntityListStage';
import RelatedEntityStage from './stages/RelatedEntityStage';

import BreadCrumbs from './BreadCrumbs';

const STAGES = {
    ENTITY: 'ENTITY',
    RELATED_ENTITY_LIST: 'RELATED_ENTITY_LIST',
    RELATED_ENTITY: 'RELATED_ENTITY'
};

const getStage = (entityId1, entityListType2, entityId2) => {
    if (!entityId1) return null;
    if (entityId2) return STAGES.RELATED_ENTITY;
    if (entityListType2 && !entityId2) return STAGES.RELATED_ENTITY_LIST;
    return STAGES.ENTITY;
};

const getStageComponent = (entityId1, entityListType2, entityId2) => {
    const stage = getStage(entityId1, entityListType2, entityId2);
    let component = null;
    switch (stage) {
        case STAGES.ENTITY:
            component = EntityStage;
            break;
        case STAGES.RELATED_ENTITY_LIST:
            component = RelatedEntityListStage;
            break;
        case STAGES.RELATED_ENTITY:
            component = RelatedEntityStage;
            break;
        default:
            break;
    }
    return component;
};

const ExternalLink = ({ onClick }) => {
    return (
        <div className="flex items-center h-full hover:bg-base-300">
            <Button
                className="border-l border-base-300 h-full px-4"
                icon={<ExternalLinkIcon className="h-6 w-6 text-base-600" />}
                onClick={onClick}
            />
        </div>
    );
};

ExternalLink.propTypes = {
    onClick: PropTypes.func.isRequired
};

const SidePanel = ({
    match,
    location,
    history,
    className,
    onClose,
    entityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2
}) => {
    const StageComponent = getStageComponent(entityId1, entityListType2, entityId2);
    const stageProps = {
        match,
        location,
        history,
        entityType1,
        entityId1,
        entityType2,
        entityListType2,
        entityId2,
        onClose
    };

    function onExternalLinkClick() {
        const stage = getStage(entityId1, entityListType2, entityId2);
        let entityType = null;
        let entityId = null;
        switch (stage) {
            case STAGES.ENTITY:
                entityType = entityType1;
                entityId = entityId1;
                break;
            case STAGES.RELATED_ENTITY_LIST:
                entityType = entityListType2;
                break;
            case STAGES.RELATED_ENTITY:
                entityType = entityType2 || entityListType2;
                entityId = entityId2;
                break;
            default:
                break;
        }
        const urlBuilder = URLService.getURL(match, location).base(entityType, entityId);
        history.push(urlBuilder.url());
    }
    return (
        <div className={className}>
            <Panel
                bodyClassName="bg-primary-100"
                headerTextComponent={
                    <BreadCrumbs
                        className="font-700 leading-normal text-base-600 uppercase tracking-wide"
                        entityType1={entityType1}
                        entityId1={entityId1}
                        entityType2={entityType2}
                        entityListType2={entityListType2}
                        entityId2={entityId2}
                    />
                }
                headerComponents={<ExternalLink onClick={onExternalLinkClick} />}
                onClose={onClose}
            >
                <StageComponent {...stageProps} />
            </Panel>
        </div>
    );
};

SidePanel.propTypes = {
    className: PropTypes.string,
    entityType1: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
    onClose: PropTypes.func
};

SidePanel.defaultProps = {
    className: '',
    entityType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null,
    onClose: null
};

export default withRouter(SidePanel);
