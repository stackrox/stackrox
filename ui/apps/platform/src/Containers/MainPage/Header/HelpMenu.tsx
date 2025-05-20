import React, { ReactElement, useState } from 'react';
import { Link } from 'react-router-dom';
import { useDispatch } from 'react-redux';
import { Divider, Dropdown, DropdownItem, DropdownList, MenuToggle } from '@patternfly/react-core';
import { QuestionCircleIcon } from '@patternfly/react-icons';

import useMetadata from 'hooks/useMetadata';
import { actions } from 'reducers/feedback';
import { apidocsPath, apidocsPathV2 } from 'routePaths';
import { getVersionedDocs } from 'utils/versioning';

function HelpMenu(): ReactElement {
    const { releaseBuild, version } = useMetadata();
    const [isHelpMenuOpen, setIsHelpMenuOpen] = useState(false);
    const dispatch = useDispatch();

    return (
        <Dropdown
            isOpen={isHelpMenuOpen}
            onOpenChange={(isOpen) => setIsHelpMenuOpen(isOpen)}
            onOpenChangeKeys={['Escape', 'Tab']}
            onSelect={() => setIsHelpMenuOpen(false)}
            popperProps={{ position: 'right' }}
            toggle={(toggleRef) => (
                <MenuToggle
                    aria-label="Help Menu"
                    ref={toggleRef}
                    variant="plain"
                    onClick={() => setIsHelpMenuOpen((wasOpen) => !wasOpen)}
                    isExpanded={isHelpMenuOpen}
                >
                    <QuestionCircleIcon />
                </MenuToggle>
            )}
        >
            <DropdownList>
                <DropdownItem>
                    <Link to={apidocsPath}>API Reference (v1)</Link>
                </DropdownItem>
                <DropdownItem>
                    <Link to={apidocsPathV2}>API Reference (v2)</Link>
                </DropdownItem>
                <DropdownItem
                    component="button"
                    onClick={() => dispatch(actions.setFeedbackModalVisibility(true))}
                >
                    Share feedback
                </DropdownItem>
                {version && (
                    <>
                        <DropdownItem
                            to={getVersionedDocs(version)}
                            isExternalLink
                            target="_blank"
                            rel="noopener noreferrer"
                        >
                            Help Center
                        </DropdownItem>
                        <Divider component="li" />
                        <DropdownItem isDisabled>
                            {`v${version}${releaseBuild ? '' : ' [DEV BUILD]'}`}
                        </DropdownItem>
                    </>
                )}
            </DropdownList>
        </Dropdown>
    );
}

export default HelpMenu;
